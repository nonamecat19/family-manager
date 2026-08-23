// Package handler implements auth.v1.AuthService over Connect.
//
// The shape of a session, in one place:
//
//   - Register creates the account. It does not sign anyone in — the client calls Login next,
//     so there is exactly one code path that mints a session.
//   - Login verifies the password with argon2id and mints an access token plus a refresh token.
//   - Refresh rotates: the presented token is marked used and a successor is issued. Presenting
//     an already-used token means it leaked, so the entire chain is revoked.
//   - Logout revokes the presented token's chain.
//
// Only the SHA-256 of a refresh token is ever stored.
package handler

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	fmauth "github.com/nnc/family-manager/libs/go/auth"
	"github.com/nnc/family-manager/libs/go/database/pgconv"
	"github.com/nnc/family-manager/libs/go/rpc"
	authv1 "github.com/nnc/family-manager/sdk/go/auth/v1"
	"github.com/nnc/family-manager/services/auth/db"
	"github.com/nnc/family-manager/services/auth/internal/password"
	"github.com/nnc/family-manager/services/auth/internal/token"
)

// Signer is the token-minting surface the handler needs.
type Signer interface {
	Sign(c token.Claims) (string, error)
	TTL() time.Duration
}

// FamilyLookup resolves which household a user belongs to. Implemented by a client of
// services/family; nil disables the lookup and every token is minted without a family_id.
type FamilyLookup interface {
	FamilyOf(ctx context.Context, userID string) (familyID string, err error)
}

type Handler struct {
	q          db.Querier
	signer     Signer
	family     FamilyLookup
	log        *slog.Logger
	hashParams password.Params
	hashGate   *password.Gate
	refreshTTL time.Duration
	now        func() time.Time
}

type Options struct {
	Queries    db.Querier
	Signer     Signer
	Family     FamilyLookup
	Log        *slog.Logger
	HashParams password.Params
	// HashGate bounds concurrent argon2id work. Nil admits everything, which is what tests
	// want and what a single-user deployment can live with.
	HashGate   *password.Gate
	RefreshTTL time.Duration
	Now        func() time.Time
}

func New(opts Options) *Handler {
	h := &Handler{
		q:          opts.Queries,
		signer:     opts.Signer,
		family:     opts.Family,
		log:        opts.Log,
		hashParams: opts.HashParams,
		hashGate:   opts.HashGate,
		refreshTTL: opts.RefreshTTL,
		now:        opts.Now,
	}
	if h.log == nil {
		h.log = slog.Default()
	}
	if h.hashParams == (password.Params{}) {
		h.hashParams = password.DefaultParams()
	}
	if h.refreshTTL == 0 {
		h.refreshTTL = 30 * 24 * time.Hour
	}
	if h.now == nil {
		h.now = time.Now
	}
	return h
}

// minPasswordLength is a floor, not a policy. Composition rules ("one symbol, one digit")
// push users toward predictable substitutions; length is what actually costs an attacker.
const minPasswordLength = 8

// Upper bounds. None of these is a policy either — they exist because every one of these
// fields arrives unauthenticated and goes somewhere that costs something: the password into
// argon2id, the email into a UNIQUE index, the name into a TEXT column with no width.
const (
	// maxPasswordBytes is far above any real passphrase. argon2id's cost does not grow with
	// input length, so this is not about hashing time; it is about not accepting a megabyte
	// of body per attempt and not storing what we refuse to bound.
	maxPasswordBytes = 1024
	// maxNameLength is in runes, not bytes: a Ukrainian name must not be worth half as much
	// as an English one.
	maxNameLength = 100
)

func (h *Handler) Register(
	ctx context.Context, req *connect.Request[authv1.RegisterRequest],
) (*connect.Response[authv1.RegisterResponse], error) {
	email := fmauth.NormalizeEmail(req.Msg.GetEmail())
	if !fmauth.LooksLikeEmail(email) {
		return nil, invalid("a valid email is required")
	}
	if len([]rune(req.Msg.GetPassword())) < minPasswordLength {
		return nil, invalid(fmt.Sprintf("password must be at least %d characters", minPasswordLength))
	}
	if len(req.Msg.GetPassword()) > maxPasswordBytes {
		return nil, invalid(fmt.Sprintf("password must be at most %d bytes", maxPasswordBytes))
	}

	name := strings.TrimSpace(req.Msg.GetName())
	if len([]rune(name)) > maxNameLength {
		return nil, invalid(fmt.Sprintf("name must be at most %d characters", maxNameLength))
	}

	var hash string
	if err := h.hashGate.Do(ctx, func() (err error) {
		hash, err = password.Hash(req.Msg.GetPassword(), h.hashParams)
		return err
	}); err != nil {
		if errors.Is(err, password.ErrBusy) {
			return nil, errBusy()
		}
		return nil, h.internal(ctx, err, "hash password")
	}

	user, err := h.q.CreateUser(ctx, db.CreateUserParams{
		Email:        email,
		Name:         name,
		PasswordHash: hash,
	})
	if err != nil {
		if isUniqueViolation(err) {
			// AlreadyExists on a public endpoint does confirm the address is registered. That
			// is unavoidable for a self-service signup form — the alternative is a flow that
			// emails the address instead, which is a product decision, not a handler one.
			return nil, connect.NewError(connect.CodeAlreadyExists,
				errors.New("that email is already registered"))
		}
		return nil, h.internal(ctx, err, "create user")
	}

	return connect.NewResponse(&authv1.RegisterResponse{
		UserId: pgconv.UUIDString(user.ID),
	}), nil
}

func (h *Handler) Login(
	ctx context.Context, req *connect.Request[authv1.LoginRequest],
) (*connect.Response[authv1.LoginResponse], error) {
	email := fmauth.NormalizeEmail(req.Msg.GetEmail())
	// Refused before the lookup and before the gate. A password no account could have is not
	// worth a database round trip, let alone 19 MiB of argon2id — and rejecting it says
	// nothing about whether the address exists, so the enumeration guarantee holds.
	if len(email) > fmauth.MaxEmailLength || len(req.Msg.GetPassword()) > maxPasswordBytes {
		return nil, errInvalidCredentials()
	}

	user, err := h.q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Hash anyway. Returning early here would make "no such user" measurably faster
			// than "wrong password", which is an account-enumeration oracle even though the
			// error text is identical. It goes through the gate for the same reason a real
			// verify does: an unknown address must not be the cheap path to hold a slot.
			_ = h.hashGate.Do(ctx, func() error {
				_, err := password.Hash(req.Msg.GetPassword(), h.hashParams)
				return err
			})
			return nil, errInvalidCredentials()
		}
		return nil, h.internal(ctx, err, "get user")
	}

	if err := h.hashGate.Do(ctx, func() error {
		return password.Verify(req.Msg.GetPassword(), user.PasswordHash)
	}); err != nil {
		if errors.Is(err, password.ErrBusy) {
			return nil, errBusy()
		}
		return nil, errInvalidCredentials()
	}

	chainID, err := newChainID()
	if err != nil {
		return nil, h.internal(ctx, err, "generate chain id")
	}

	tokens, err := h.mintSession(ctx, user, chainID)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&authv1.LoginResponse{
		AccessToken:  tokens.access,
		RefreshToken: tokens.refresh,
		ExpiresIn:    tokens.expiresIn,
	}), nil
}

func (h *Handler) Refresh(
	ctx context.Context, req *connect.Request[authv1.RefreshRequest],
) (*connect.Response[authv1.RefreshResponse], error) {
	presented := strings.TrimSpace(req.Msg.GetRefreshToken())
	if presented == "" {
		return nil, invalid("refresh_token is required")
	}

	row, err := h.q.GetRefreshToken(ctx, hashToken(presented))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errInvalidRefresh()
		}
		return nil, h.internal(ctx, err, "get refresh token")
	}

	if row.RevokedAt.Valid {
		return nil, errInvalidRefresh()
	}
	if row.UsedAt.Valid {
		// Replay. Either the client raced itself or the token was stolen; we cannot tell, so
		// we assume the worse case and kill every descendant of this chain.
		if _, err := h.q.RevokeChain(ctx, row.ChainID); err != nil {
			h.log.ErrorContext(ctx, "revoke chain after replay",
				slog.String("error", err.Error()))
		}
		h.log.WarnContext(ctx, "refresh token replayed; chain revoked",
			slog.String("user_id", pgconv.UUIDString(row.UserID)))
		return nil, errInvalidRefresh()
	}
	if row.ExpiresAt.Valid && !row.ExpiresAt.Time.After(h.now()) {
		return nil, errInvalidRefresh()
	}

	// Marking used is the concurrency guard: two refreshes racing on one token, only one
	// updates a row, and the loser is treated as a replay on its next attempt.
	spent, err := h.q.MarkRefreshTokenUsed(ctx, row.ID)
	if err != nil {
		return nil, h.internal(ctx, err, "mark refresh token used")
	}
	if spent == 0 {
		return nil, errInvalidRefresh()
	}

	user, err := h.q.GetUserByID(ctx, row.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errInvalidRefresh()
		}
		return nil, h.internal(ctx, err, "get user")
	}

	// Same chain: rotation replaces a token, it does not start a new session.
	tokens, err := h.mintSession(ctx, user, row.ChainID)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&authv1.RefreshResponse{
		AccessToken:  tokens.access,
		RefreshToken: tokens.refresh,
		ExpiresIn:    tokens.expiresIn,
	}), nil
}

func (h *Handler) Logout(
	ctx context.Context, req *connect.Request[authv1.LogoutRequest],
) (*connect.Response[authv1.LogoutResponse], error) {
	presented := strings.TrimSpace(req.Msg.GetRefreshToken())
	if presented == "" {
		return nil, invalid("refresh_token is required")
	}

	row, err := h.q.GetRefreshToken(ctx, hashToken(presented))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Logging out with a token we do not recognise ends in the same place the caller
			// wanted: no session. Reporting an error would only tell an attacker which
			// tokens exist.
			return connect.NewResponse(&authv1.LogoutResponse{}), nil
		}
		return nil, h.internal(ctx, err, "get refresh token")
	}

	if _, err := h.q.RevokeChain(ctx, row.ChainID); err != nil {
		return nil, h.internal(ctx, err, "revoke chain")
	}
	return connect.NewResponse(&authv1.LogoutResponse{}), nil
}

/* ----------------------------------------------------------------- internals */

type session struct {
	access    string
	refresh   string
	expiresIn int64
}

// mintSession issues an access token and a refresh token belonging to chainID.
func (h *Handler) mintSession(ctx context.Context, user db.User, chainID pgtype.UUID) (session, error) {
	familyID := h.familyOf(ctx, pgconv.UUIDString(user.ID))

	access, err := h.signer.Sign(token.Claims{
		UserID:   pgconv.UUIDString(user.ID),
		Email:    user.Email,
		FamilyID: familyID,
	})
	if err != nil {
		return session{}, h.internal(ctx, err, "sign access token")
	}

	refresh, err := newRefreshToken()
	if err != nil {
		return session{}, h.internal(ctx, err, "generate refresh token")
	}

	if _, err := h.q.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		UserID:    user.ID,
		TokenHash: hashToken(refresh),
		ChainID:   chainID,
		ExpiresAt: pgconv.TimestampFrom(h.now().Add(h.refreshTTL)),
	}); err != nil {
		// No rows means the query's guard fired: the chain was revoked between this
		// refresh's check and its insert. Refusing is the whole point — resurrecting a
		// chain a replay just killed would undo the theft response.
		if errors.Is(err, pgx.ErrNoRows) {
			return session{}, errInvalidRefresh()
		}
		return session{}, h.internal(ctx, err, "store refresh token")
	}

	return session{
		access:    access,
		refresh:   refresh,
		expiresIn: int64(h.signer.TTL().Seconds()),
	}, nil
}

// familyOf resolves the family_id claim. A failure is logged and swallowed: a household
// lookup outage must not stop people signing in, and a token without the claim degrades to
// "show onboarding" rather than to wrong authorization.
func (h *Handler) familyOf(ctx context.Context, userID string) string {
	if h.family == nil {
		return ""
	}
	familyID, err := h.family.FamilyOf(ctx, userID)
	if err != nil {
		h.log.WarnContext(ctx, "family lookup failed; minting a token without family_id",
			slog.String("user_id", userID), slog.String("error", err.Error()))
		return ""
	}
	return familyID
}

func newRefreshToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashToken(t string) string {
	sum := sha256.Sum256([]byte(t))
	return hex.EncodeToString(sum[:])
}

// newChainID mints the v4 UUID that identifies one refresh chain.
//
// It returns the error rather than a zero value on purpose. The previous version fell back to
// "", which pgconv turns into an invalid (NULL) pgtype.UUID — the insert would then succeed
// with chain_id NULL, and RevokeChain(NULL) matches no rows. A CSPRNG failure would have
// quietly produced sessions that survive the replay response instead of being killed by it.
func newChainID() (pgtype.UUID, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return pgtype.UUID{}, fmt.Errorf("read random bytes: %w", err)
	}
	buf[6] = (buf[6] & 0x0f) | 0x40 // version 4
	buf[8] = (buf[8] & 0x3f) | 0x80 // variant 10
	return pgtype.UUID{Bytes: buf, Valid: true}, nil
}

// errInvalidCredentials is the single answer to a bad email and a bad password alike:
// distinguishing them tells an attacker which addresses are registered.
func errInvalidCredentials() error {
	return connect.NewError(connect.CodeUnauthenticated, errors.New("invalid email or password"))
}

// errBusy is Unavailable rather than ResourceExhausted: the caller did nothing wrong and
// should retry, which is exactly what Unavailable tells a Connect client.
func errBusy() error {
	return connect.NewError(connect.CodeUnavailable,
		errors.New("too many sign-in attempts in flight; try again"))
}

func errInvalidRefresh() error {
	return connect.NewError(connect.CodeUnauthenticated, errors.New("invalid refresh token"))
}

func invalid(msg string) error {
	return connect.NewError(connect.CodeInvalidArgument, errors.New(msg))
}

// internal hands the cause to the log and an opaque reference to the caller. Register, Login
// and Refresh are unauthenticated, so a pgx error rendered into the response body is readable
// by anyone who can reach the port.
func (h *Handler) internal(ctx context.Context, err error, what string) error {
	return rpc.Internal(ctx, h.log, err, what)
}

// isUniqueViolation reports whether err is Postgres SQLSTATE 23505.
func isUniqueViolation(err error) bool {
	var pgErr interface{ SQLState() string }
	return errors.As(err, &pgErr) && pgErr.SQLState() == "23505"
}
