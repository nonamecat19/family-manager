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
	"net/netip"
	"strings"
	"sync/atomic"
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
	"github.com/nnc/family-manager/services/auth/internal/ratelimit"
	"github.com/nnc/family-manager/services/auth/internal/throttle"
	"github.com/nnc/family-manager/services/auth/internal/token"
)

type Signer interface {
	Sign(c token.Claims) (string, error)
	TTL() time.Duration
	ChainOf(ctx context.Context, signed string) (string, error)
}

type FamilyLookup interface {
	FamilyOf(ctx context.Context, userID string) (familyID string, err error)
	LocaleOf(ctx context.Context, userID string) (locale string, err error)
}

type Tx interface {
	InTx(ctx context.Context, fn func(q db.Querier) error) error
}

type Handler struct {
	q          db.Querier
	tx         Tx
	signer     Signer
	family     FamilyLookup
	log        *slog.Logger
	hashParams password.Params
	hashGate   *password.Gate
	throttle   *throttle.Throttle
	decisions  *throttle.Throttle
	starts     *ratelimit.Limiter
	maxPending int64
	refreshTTL time.Duration
	now        func() time.Time

	trustedProxies  []netip.Prefix
	pendingWarnedAt atomic.Int64
}

type Options struct {
	Queries    db.Querier
	Tx         Tx
	Signer     Signer
	Family     FamilyLookup
	Log        *slog.Logger
	HashParams password.Params
	HashGate   *password.Gate
	Throttle   *throttle.Throttle
	Decisions  *throttle.Throttle
	Starts     *ratelimit.Limiter
	MaxPending int64
	RefreshTTL time.Duration
	Now        func() time.Time

	TrustedProxies []netip.Prefix
}

func New(opts Options) *Handler {
	h := &Handler{
		q:          opts.Queries,
		tx:         opts.Tx,
		signer:     opts.Signer,
		family:     opts.Family,
		log:        opts.Log,
		hashParams: opts.HashParams,
		hashGate:   opts.HashGate,
		throttle:   opts.Throttle,
		decisions:  opts.Decisions,
		starts:     opts.Starts,
		maxPending: opts.MaxPending,
		refreshTTL: opts.RefreshTTL,
		now:        opts.Now,

		trustedProxies: opts.TrustedProxies,
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
	if h.tx == nil {
		h.tx = withoutTx{h.q}
	}
	if h.decisions == nil {
		h.decisions = throttle.New(throttle.DefaultParams(), nil)
	}
	return h
}

type withoutTx struct{ q db.Querier }

func (w withoutTx) InTx(_ context.Context, fn func(db.Querier) error) error { return fn(w.q) }

const minPasswordLength = 8

const (
	maxPasswordBytes = 1024
	maxNameLength    = 100
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
	if len(email) > fmauth.MaxEmailLength || len(req.Msg.GetPassword()) > maxPasswordBytes {
		return nil, errInvalidCredentials()
	}

	if wait := h.throttle.Retry(email); wait > 0 {
		return nil, errTooManyAttempts(wait)
	}

	user, err := h.q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			_ = h.hashGate.Do(ctx, func() error {
				_, err := password.Hash(req.Msg.GetPassword(), h.hashParams)
				return err
			})
			h.throttle.Failed(email)
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
		h.throttle.Failed(email)
		return nil, errInvalidCredentials()
	}
	h.throttle.Succeeded(email)

	chainID, err := newChainID()
	if err != nil {
		return nil, h.internal(ctx, err, "generate chain id")
	}

	tokens, err := h.mintSession(ctx, user, chainID, true)
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
		if err := h.tx.InTx(ctx, func(q db.Querier) error {
			return revokeChain(ctx, q, row.ChainID)
		}); err != nil {
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

	tokens, err := h.mintSession(ctx, user, row.ChainID, false)
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
			return connect.NewResponse(&authv1.LogoutResponse{}), nil
		}
		return nil, h.internal(ctx, err, "get refresh token")
	}

	if err := h.tx.InTx(ctx, func(q db.Querier) error {
		return revokeChain(ctx, q, row.ChainID)
	}); err != nil {
		return nil, h.internal(ctx, err, "revoke chain")
	}
	return connect.NewResponse(&authv1.LogoutResponse{}), nil
}

type session struct {
	access    string
	refresh   string
	expiresIn int64
}

func (h *Handler) mintSession(
	ctx context.Context, user db.User, chainID pgtype.UUID, fresh bool,
) (session, error) {
	userID := pgconv.UUIDString(user.ID)
	familyID := h.familyOf(ctx, userID)
	locale := h.localeOf(ctx, userID)

	access, err := h.signer.Sign(token.Claims{
		UserID:   userID,
		Email:    user.Email,
		FamilyID: familyID,
		Locale:   locale,
		ChainID:  pgconv.UUIDString(chainID),
	})
	if err != nil {
		return session{}, h.internal(ctx, err, "sign access token")
	}

	refresh, err := newRefreshToken()
	if err != nil {
		return session{}, h.internal(ctx, err, "generate refresh token")
	}

	err = h.tx.InTx(ctx, func(q db.Querier) error {
		revokedAt, err := lockChain(ctx, q, chainID, fresh)
		if errors.Is(err, pgx.ErrNoRows) {
			return errInvalidRefresh()
		}
		if err != nil {
			return err
		}
		if revokedAt.Valid {
			return errInvalidRefresh()
		}
		if _, err := q.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
			UserID:    user.ID,
			TokenHash: hashToken(refresh),
			ChainID:   chainID,
			ExpiresAt: pgconv.TimestampFrom(h.now().Add(h.refreshTTL)),
		}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return errInvalidRefresh()
			}
			return err
		}
		return nil
	})
	if err != nil {
		var cerr *connect.Error
		if errors.As(err, &cerr) {
			return session{}, err
		}
		return session{}, h.internal(ctx, err, "store refresh token")
	}

	return session{
		access:    access,
		refresh:   refresh,
		expiresIn: int64(h.signer.TTL().Seconds()),
	}, nil
}

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

func (h *Handler) localeOf(ctx context.Context, userID string) string {
	if h.family == nil {
		return ""
	}
	locale, err := h.family.LocaleOf(ctx, userID)
	if err != nil {
		h.log.WarnContext(ctx, "settings lookup failed; minting a token without locale",
			slog.String("user_id", userID), slog.String("error", err.Error()))
		return ""
	}
	return locale
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

func newChainID() (pgtype.UUID, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return pgtype.UUID{}, fmt.Errorf("read random bytes: %w", err)
	}
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80
	return pgtype.UUID{Bytes: buf, Valid: true}, nil
}

func errInvalidCredentials() error {
	return connect.NewError(connect.CodeUnauthenticated, errors.New("invalid email or password"))
}

func errTooManyAttempts(wait time.Duration) error {
	return connect.NewError(connect.CodeResourceExhausted,
		fmt.Errorf("too many failed sign-in attempts; try again in %s", wait.Round(time.Second)))
}

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

func (h *Handler) internal(ctx context.Context, err error, what string) error {
	return rpc.Internal(ctx, h.log, err, what)
}

func isUniqueViolation(err error) bool {
	var pgErr interface{ SQLState() string }
	return errors.As(err, &pgErr) && pgErr.SQLState() == "23505"
}
