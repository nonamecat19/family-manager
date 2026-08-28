package handler

import (
	"context"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jackc/pgx/v5/pgtype"

	fmauth "github.com/nnc/family-manager/libs/go/auth"
	"github.com/nnc/family-manager/libs/go/database/pgconv"
	authv1 "github.com/nnc/family-manager/sdk/go/auth/v1"
	"github.com/nnc/family-manager/services/auth/db"
	"github.com/nnc/family-manager/services/auth/internal/password"
	"github.com/nnc/family-manager/services/auth/internal/throttle"
)

type fixture struct {
	h      *Handler
	store  *fakeStore
	signer *stubSigner
	family *stubFamily
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	store := newFakeStore()
	signer := &stubSigner{}
	fam := &stubFamily{}

	// Cheap KDF parameters: these tests exercise the session rules, not argon2's cost.
	params := password.DefaultParams()
	params.Memory = 64
	params.Iterations = 1

	return &fixture{
		h: New(Options{
			Queries:    store,
			Signer:     signer,
			Family:     fam,
			HashParams: params,
			RefreshTTL: 30 * 24 * time.Hour,
		}),
		store:  store,
		signer: signer,
		family: fam,
	}
}

func (f *fixture) register(t *testing.T, email, pw string) string {
	t.Helper()
	res, err := f.h.Register(context.Background(),
		connect.NewRequest(&authv1.RegisterRequest{Email: email, Password: pw, Name: "Test"}))
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	return res.Msg.GetUserId()
}

func (f *fixture) login(t *testing.T, email, pw string) *authv1.LoginResponse {
	t.Helper()
	res, err := f.h.Login(context.Background(),
		connect.NewRequest(&authv1.LoginRequest{Email: email, Password: pw}))
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	return res.Msg
}

/* ----------------------------------------------------------------- register */

func TestRegisterThenLogin(t *testing.T) {
	f := newFixture(t)
	userID := f.register(t, "ada@example.test", "correct horse")

	if userID == "" {
		t.Fatal("Register returned no user id")
	}

	res := f.login(t, "ada@example.test", "correct horse")
	if res.GetAccessToken() == "" || res.GetRefreshToken() == "" {
		t.Fatal("Login returned an empty token")
	}
	if res.GetExpiresIn() != int64((15 * time.Minute).Seconds()) {
		t.Errorf("expires_in = %d, want 900", res.GetExpiresIn())
	}
}

func TestRegisterDoesNotStoreThePlaintextPassword(t *testing.T) {
	f := newFixture(t)
	f.register(t, "ada@example.test", "correct horse")

	stored := f.store.users["ada@example.test"].PasswordHash
	if strings.Contains(stored, "correct horse") {
		t.Fatal("the plaintext password appears in the stored hash")
	}
	if !strings.HasPrefix(stored, "$argon2id$") {
		t.Errorf("stored value is not an argon2id hash: %q", stored)
	}
}

func TestRegisterNormalisesTheEmail(t *testing.T) {
	f := newFixture(t)
	f.register(t, "  Ada@Example.TEST  ", "correct horse")

	if _, ok := f.store.users["ada@example.test"]; !ok {
		t.Fatalf("email was not normalised: %v", keysOf(f.store.users))
	}
	// And the normalised address is what logs in, whatever case the user types.
	f.login(t, "ADA@example.test", "correct horse")
}

func TestRegisterRejectsADuplicateEmail(t *testing.T) {
	f := newFixture(t)
	f.register(t, "ada@example.test", "correct horse")

	_, err := f.h.Register(context.Background(), connect.NewRequest(&authv1.RegisterRequest{
		Email: "Ada@example.test", Password: "another one",
	}))
	if connect.CodeOf(err) != connect.CodeAlreadyExists {
		t.Fatalf("code = %v, want already_exists", connect.CodeOf(err))
	}
}

func TestRegisterValidatesInput(t *testing.T) {
	f := newFixture(t)

	for _, email := range []string{"", "not-an-email", "@example.test", "ada@", "a b@c.test", "ada@test"} {
		_, err := f.h.Register(context.Background(), connect.NewRequest(&authv1.RegisterRequest{
			Email: email, Password: "correct horse",
		}))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Errorf("email %q: code = %v, want invalid_argument", email, connect.CodeOf(err))
		}
	}

	_, err := f.h.Register(context.Background(), connect.NewRequest(&authv1.RegisterRequest{
		Email: "ada@example.test", Password: "short",
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("short password: code = %v, want invalid_argument", connect.CodeOf(err))
	}
}

/* -------------------------------------------------------------------- login */

func TestLoginRejectsTheWrongPassword(t *testing.T) {
	f := newFixture(t)
	f.register(t, "ada@example.test", "correct horse")

	_, err := f.h.Login(context.Background(),
		connect.NewRequest(&authv1.LoginRequest{Email: "ada@example.test", Password: "wrong"}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("code = %v, want unauthenticated", connect.CodeOf(err))
	}
}

// The error for an unknown account and a wrong password must be indistinguishable, or the
// endpoint tells an attacker which addresses are registered.
func TestLoginDoesNotRevealWhetherAnAccountExists(t *testing.T) {
	f := newFixture(t)
	f.register(t, "ada@example.test", "correct horse")

	_, wrongPassword := f.h.Login(context.Background(),
		connect.NewRequest(&authv1.LoginRequest{Email: "ada@example.test", Password: "wrong"}))
	_, noSuchUser := f.h.Login(context.Background(),
		connect.NewRequest(&authv1.LoginRequest{Email: "nobody@example.test", Password: "wrong"}))

	if connect.CodeOf(wrongPassword) != connect.CodeOf(noSuchUser) {
		t.Errorf("codes differ: %v vs %v", connect.CodeOf(wrongPassword), connect.CodeOf(noSuchUser))
	}
	if wrongPassword.Error() != noSuchUser.Error() {
		t.Errorf("messages differ:\n  %q\n  %q", wrongPassword, noSuchUser)
	}
}

func TestLoginStampsTheFamilyClaim(t *testing.T) {
	f := newFixture(t)
	f.family.familyID = "fam-1"
	f.register(t, "ada@example.test", "correct horse")
	f.login(t, "ada@example.test", "correct horse")

	if got := f.signer.last().FamilyID; got != "fam-1" {
		t.Errorf("family_id claim = %q, want fam-1", got)
	}
	if got := f.signer.last().Email; got != "ada@example.test" {
		t.Errorf("email claim = %q", got)
	}
}

// A household lookup outage must not stop people signing in: the token is minted without the
// claim and the app shows onboarding.
func TestLoginSucceedsWhenTheFamilyLookupFails(t *testing.T) {
	f := newFixture(t)
	f.family.err = errBoom
	f.register(t, "ada@example.test", "correct horse")

	res := f.login(t, "ada@example.test", "correct horse")
	if res.GetAccessToken() == "" {
		t.Fatal("login failed because the family lookup did")
	}
	if got := f.signer.last().FamilyID; got != "" {
		t.Errorf("family_id = %q, want empty", got)
	}
}

func TestRefreshTokenIsStoredOnlyAsAHash(t *testing.T) {
	f := newFixture(t)
	f.register(t, "ada@example.test", "correct horse")
	res := f.login(t, "ada@example.test", "correct horse")

	for hash := range f.store.tokens {
		if hash == res.GetRefreshToken() {
			t.Fatal("the refresh token is stored in plaintext")
		}
		if hash != hashToken(res.GetRefreshToken()) {
			t.Errorf("stored hash does not match the issued token")
		}
	}
}

func TestEachLoginStartsItsOwnChain(t *testing.T) {
	f := newFixture(t)
	f.register(t, "ada@example.test", "correct horse")

	first := f.login(t, "ada@example.test", "correct horse")
	second := f.login(t, "ada@example.test", "correct horse")

	a := f.store.tokens[hashToken(first.GetRefreshToken())]
	b := f.store.tokens[hashToken(second.GetRefreshToken())]
	if pgUUID(a.ChainID) == pgUUID(b.ChainID) {
		t.Fatal("two logins share a rotation chain; logging out of one would kill the other")
	}
}

/* ------------------------------------------------------------------ refresh */

func TestRefreshRotatesTheToken(t *testing.T) {
	f := newFixture(t)
	f.family.familyID = "fam-1"
	f.register(t, "ada@example.test", "correct horse")
	first := f.login(t, "ada@example.test", "correct horse")

	res, err := f.h.Refresh(context.Background(),
		connect.NewRequest(&authv1.RefreshRequest{RefreshToken: first.GetRefreshToken()}))
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}

	if res.Msg.GetRefreshToken() == first.GetRefreshToken() {
		t.Fatal("Refresh returned the same refresh token; it must rotate")
	}
	if res.Msg.GetAccessToken() == "" {
		t.Fatal("Refresh returned no access token")
	}

	// The successor belongs to the same session.
	old := f.store.tokens[hashToken(first.GetRefreshToken())]
	fresh := f.store.tokens[hashToken(res.Msg.GetRefreshToken())]
	if pgUUID(old.ChainID) != pgUUID(fresh.ChainID) {
		t.Error("rotation started a new chain; it should continue the session")
	}
	if !old.UsedAt.Valid {
		t.Error("the presented token was not marked used")
	}

	// Claims are re-resolved on rotation, so joining a household takes effect on refresh.
	if f.family.calls != 2 {
		t.Errorf("family lookups = %d, want 2 (login and refresh)", f.family.calls)
	}
}

// Replay is the case that matters: a stolen token used after the victim's client rotated
// must invalidate the whole chain, not just the stolen copy.
func TestReplayRevokesTheWholeChain(t *testing.T) {
	f := newFixture(t)
	f.register(t, "ada@example.test", "correct horse")
	first := f.login(t, "ada@example.test", "correct horse")

	rotated, err := f.h.Refresh(context.Background(),
		connect.NewRequest(&authv1.RefreshRequest{RefreshToken: first.GetRefreshToken()}))
	if err != nil {
		t.Fatalf("first Refresh: %v", err)
	}

	// The thief presents the original again.
	_, replay := f.h.Refresh(context.Background(),
		connect.NewRequest(&authv1.RefreshRequest{RefreshToken: first.GetRefreshToken()}))
	if connect.CodeOf(replay) != connect.CodeUnauthenticated {
		t.Fatalf("replay code = %v, want unauthenticated", connect.CodeOf(replay))
	}

	// And the victim's freshly rotated token is dead too.
	_, victim := f.h.Refresh(context.Background(),
		connect.NewRequest(&authv1.RefreshRequest{RefreshToken: rotated.Msg.GetRefreshToken()}))
	if connect.CodeOf(victim) != connect.CodeUnauthenticated {
		t.Fatalf("the chain was not revoked: victim refresh returned %v", connect.CodeOf(victim))
	}
}

func TestRefreshRejectsUnknownExpiredAndRevokedTokens(t *testing.T) {
	f := newFixture(t)
	f.register(t, "ada@example.test", "correct horse")

	// Unknown.
	_, err := f.h.Refresh(context.Background(),
		connect.NewRequest(&authv1.RefreshRequest{RefreshToken: "never-issued"}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Errorf("unknown token: code = %v", connect.CodeOf(err))
	}

	// Empty.
	_, err = f.h.Refresh(context.Background(),
		connect.NewRequest(&authv1.RefreshRequest{RefreshToken: "   "}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("empty token: code = %v, want invalid_argument", connect.CodeOf(err))
	}

	// Expired: mint with the clock a year back.
	f.h.now = func() time.Time { return time.Now().Add(-365 * 24 * time.Hour) }
	stale := f.login(t, "ada@example.test", "correct horse")
	f.h.now = time.Now

	_, err = f.h.Refresh(context.Background(),
		connect.NewRequest(&authv1.RefreshRequest{RefreshToken: stale.GetRefreshToken()}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Errorf("expired token: code = %v", connect.CodeOf(err))
	}
}

/* ------------------------------------------------------------------- logout */

func TestLogoutRevokesTheChain(t *testing.T) {
	f := newFixture(t)
	f.register(t, "ada@example.test", "correct horse")
	session := f.login(t, "ada@example.test", "correct horse")

	if _, err := f.h.Logout(context.Background(),
		connect.NewRequest(&authv1.LogoutRequest{RefreshToken: session.GetRefreshToken()})); err != nil {
		t.Fatalf("Logout: %v", err)
	}

	_, err := f.h.Refresh(context.Background(),
		connect.NewRequest(&authv1.RefreshRequest{RefreshToken: session.GetRefreshToken()}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("the token still works after logout: %v", connect.CodeOf(err))
	}
}

func TestLogoutOfOneSessionLeavesTheOtherAlive(t *testing.T) {
	f := newFixture(t)
	f.register(t, "ada@example.test", "correct horse")
	phone := f.login(t, "ada@example.test", "correct horse")
	laptop := f.login(t, "ada@example.test", "correct horse")

	if _, err := f.h.Logout(context.Background(),
		connect.NewRequest(&authv1.LogoutRequest{RefreshToken: phone.GetRefreshToken()})); err != nil {
		t.Fatalf("Logout: %v", err)
	}

	if _, err := f.h.Refresh(context.Background(),
		connect.NewRequest(&authv1.RefreshRequest{RefreshToken: laptop.GetRefreshToken()})); err != nil {
		t.Fatalf("logging out on one device killed the other: %v", err)
	}
}

// An unknown token ends where the caller wanted — no session — and reporting an error would
// only tell an attacker which tokens exist.
func TestLogoutWithAnUnknownTokenSucceeds(t *testing.T) {
	f := newFixture(t)
	if _, err := f.h.Logout(context.Background(),
		connect.NewRequest(&authv1.LogoutRequest{RefreshToken: "never-issued"})); err != nil {
		t.Fatalf("Logout: %v", err)
	}
}

/* --------------------------------------------------------------------- misc */

func TestStoreFailureBecomesInternal(t *testing.T) {
	f := newFixture(t)
	f.store.failOn["CreateUser"] = errBoom

	_, err := f.h.Register(context.Background(), connect.NewRequest(&authv1.RegisterRequest{
		Email: "ada@example.test", Password: "correct horse",
	}))
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("code = %v, want internal", connect.CodeOf(err))
	}
	// Register is unauthenticated: whatever ends up in this message is readable by anyone who
	// can reach the port, so the cause must not be in it.
	if strings.Contains(err.Error(), errBoom.Error()) {
		t.Fatalf("wire message leaked the cause: %q", err.Error())
	}
}

func TestNewChainIDIsValidVersion4AndUnique(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		id, err := newChainID()
		if err != nil {
			t.Fatalf("newChainID() error = %v", err)
		}
		// Valid matters as much as the version: an invalid pgtype.UUID is written as NULL,
		// and a NULL chain_id is a chain RevokeChain can never revoke.
		if !id.Valid {
			t.Fatal("newChainID() returned an invalid (NULL) uuid")
		}
		s := pgconv.UUIDString(id)
		if len(s) != 36 || s[14] != '4' {
			t.Fatalf("newChainID() = %q, want a v4 uuid", s)
		}
		if seen[s] {
			t.Fatalf("newChainID() repeated %q", s)
		}
		seen[s] = true
	}
}

// keysOf renders the store's email keys for a failure message.
func keysOf(m map[string]db.User) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// pgUUID renders a uuid for comparison in assertions.
func pgUUID(u pgtype.UUID) string { return pgconv.UUIDString(u) }

// The race the guarded insert closes: a refresh that already passed the "chain alive?"
// check must not resurrect a chain a concurrent replay revoked in the meantime.
func TestRefreshCannotResurrectARevokedChain(t *testing.T) {
	f := newFixture(t)
	f.register(t, "ada@example.test", "correct horse")
	session := f.login(t, "ada@example.test", "correct horse")

	// Rotate once so the chain holds more than one row — the shape a real chain has when a
	// replay and a legitimate refresh race.
	rotated, err := f.h.Refresh(context.Background(),
		connect.NewRequest(&authv1.RefreshRequest{RefreshToken: session.GetRefreshToken()}))
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}

	newest := hashToken(rotated.Msg.GetRefreshToken())
	row := f.store.tokens[newest]

	// The replay handler has just revoked the chain.
	if _, err := f.store.RevokeChain(context.Background(), row.ChainID); err != nil {
		t.Fatalf("RevokeChain: %v", err)
	}
	// Stand where the loser of the race stands: it read its own row before the revocation,
	// so its early checks pass and the insert is what has to refuse.
	row.RevokedAt = pgtype.Timestamptz{}
	row.UsedAt = pgtype.Timestamptz{}
	f.store.tokens[newest] = row

	_, err = f.h.Refresh(context.Background(),
		connect.NewRequest(&authv1.RefreshRequest{RefreshToken: rotated.Msg.GetRefreshToken()}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("code = %v, want unauthenticated: a revoked chain accepted a successor",
			connect.CodeOf(err))
	}
}

func TestRegisterRejectsOversizeFields(t *testing.T) {
	cases := map[string]*authv1.RegisterRequest{
		"password": {
			Email:    "ada@example.test",
			Password: strings.Repeat("x", maxPasswordBytes+1),
		},
		"email": {
			Email:    strings.Repeat("a", fmauth.MaxEmailLength) + "@example.test",
			Password: "correct horse",
		},
		"name": {
			Email:    "ada@example.test",
			Password: "correct horse",
			Name:     strings.Repeat("я", maxNameLength+1),
		},
	}
	for field, req := range cases {
		t.Run(field, func(t *testing.T) {
			f := newFixture(t)
			_, err := f.h.Register(context.Background(), connect.NewRequest(req))
			if connect.CodeOf(err) != connect.CodeInvalidArgument {
				t.Fatalf("code = %v, want invalid_argument (err=%v)", connect.CodeOf(err), err)
			}
		})
	}
}

// The name bound is in runes: a Cyrillic name of the same length as a Latin one must be
// equally acceptable, even though it is twice the bytes.
func TestRegisterAcceptsAMaxLengthCyrillicName(t *testing.T) {
	f := newFixture(t)
	_, err := f.h.Register(context.Background(), connect.NewRequest(&authv1.RegisterRequest{
		Email:    "ada@example.test",
		Password: "correct horse",
		Name:     strings.Repeat("я", maxNameLength),
	}))
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
}

// An oversize password must not reach the store or the hasher, and must be indistinguishable
// from any other failed sign-in.
func TestLoginRejectsAnOversizePasswordWithoutALookup(t *testing.T) {
	f := newFixture(t)
	f.store.failOn["GetUserByEmail"] = errBoom

	_, err := f.h.Login(context.Background(), connect.NewRequest(&authv1.LoginRequest{
		Email:    "ada@example.test",
		Password: strings.Repeat("x", maxPasswordBytes+1),
	}))
	// errBoom would surface as internal if the lookup had run.
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("code = %v, want unauthenticated (err=%v)", connect.CodeOf(err), err)
	}
}

// Eight characters is the right floor and is not, on its own, a defence against someone trying
// passwords as fast as the service will answer.
func TestLoginLocksOutAfterRepeatedFailures(t *testing.T) {
	f := newFixture(t)
	th := throttle.New(throttle.Params{
		Threshold: 3, Base: time.Minute, Max: time.Minute, Forget: time.Hour,
	}, nil)
	f.h.throttle = th

	f.register(t, "ada@example.test", "correct horse")

	wrong := connect.NewRequest(&authv1.LoginRequest{
		Email: "ada@example.test", Password: "not the password",
	})
	for i := 0; i < 3; i++ {
		if _, err := f.h.Login(context.Background(), wrong); connect.CodeOf(err) != connect.CodeUnauthenticated {
			t.Fatalf("attempt %d code = %v, want unauthenticated", i+1, connect.CodeOf(err))
		}
	}

	_, err := f.h.Login(context.Background(), wrong)
	if connect.CodeOf(err) != connect.CodeResourceExhausted {
		t.Fatalf("code = %v, want resource_exhausted (err=%v)", connect.CodeOf(err), err)
	}

	// Locked means locked: the correct password is refused too, or the lockout is no defence.
	_, err = f.h.Login(context.Background(), connect.NewRequest(&authv1.LoginRequest{
		Email: "ada@example.test", Password: "correct horse",
	}))
	if connect.CodeOf(err) != connect.CodeResourceExhausted {
		t.Fatalf("code = %v, want resource_exhausted while locked", connect.CodeOf(err))
	}
}

// A person mistyping twice and then getting it right must not be counting down to a lockout.
func TestLoginSuccessClearsTheFailureCount(t *testing.T) {
	f := newFixture(t)
	f.h.throttle = throttle.New(throttle.Params{
		Threshold: 3, Base: time.Minute, Max: time.Minute, Forget: time.Hour,
	}, nil)

	f.register(t, "ada@example.test", "correct horse")

	for i := 0; i < 2; i++ {
		_, _ = f.h.Login(context.Background(), connect.NewRequest(&authv1.LoginRequest{
			Email: "ada@example.test", Password: "wrong",
		}))
	}
	if _, err := f.h.Login(context.Background(), connect.NewRequest(&authv1.LoginRequest{
		Email: "ada@example.test", Password: "correct horse",
	})); err != nil {
		t.Fatalf("Login with the right password: %v", err)
	}

	// The counter is back to zero, so two more failures are still under the threshold.
	for i := 0; i < 2; i++ {
		_, err := f.h.Login(context.Background(), connect.NewRequest(&authv1.LoginRequest{
			Email: "ada@example.test", Password: "wrong",
		}))
		if connect.CodeOf(err) != connect.CodeUnauthenticated {
			t.Fatalf("attempt %d code = %v, want unauthenticated", i+1, connect.CodeOf(err))
		}
	}
}

// An address nobody registered must be throttled too: otherwise it is the unthrottled way to
// probe, and which addresses start refusing leaks which ones exist.
func TestLoginThrottlesUnknownAddressesToo(t *testing.T) {
	f := newFixture(t)
	f.h.throttle = throttle.New(throttle.Params{
		Threshold: 2, Base: time.Minute, Max: time.Minute, Forget: time.Hour,
	}, nil)

	req := connect.NewRequest(&authv1.LoginRequest{
		Email: "nobody@example.test", Password: "guess",
	})
	for i := 0; i < 2; i++ {
		_, _ = f.h.Login(context.Background(), req)
	}
	if _, err := f.h.Login(context.Background(), req); connect.CodeOf(err) != connect.CodeResourceExhausted {
		t.Fatalf("code = %v, want resource_exhausted", connect.CodeOf(err))
	}
}
