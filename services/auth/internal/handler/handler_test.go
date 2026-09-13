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

func TestLoginRejectsTheWrongPassword(t *testing.T) {
	f := newFixture(t)
	f.register(t, "ada@example.test", "correct horse")

	_, err := f.h.Login(context.Background(),
		connect.NewRequest(&authv1.LoginRequest{Email: "ada@example.test", Password: "wrong"}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("code = %v, want unauthenticated", connect.CodeOf(err))
	}
}

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

	old := f.store.tokens[hashToken(first.GetRefreshToken())]
	fresh := f.store.tokens[hashToken(res.Msg.GetRefreshToken())]
	if pgUUID(old.ChainID) != pgUUID(fresh.ChainID) {
		t.Error("rotation started a new chain; it should continue the session")
	}
	if !old.UsedAt.Valid {
		t.Error("the presented token was not marked used")
	}

	if f.family.calls != 2 {
		t.Errorf("family lookups = %d, want 2 (login and refresh)", f.family.calls)
	}
}

func TestReplayRevokesTheWholeChain(t *testing.T) {
	f := newFixture(t)
	f.register(t, "ada@example.test", "correct horse")
	first := f.login(t, "ada@example.test", "correct horse")

	rotated, err := f.h.Refresh(context.Background(),
		connect.NewRequest(&authv1.RefreshRequest{RefreshToken: first.GetRefreshToken()}))
	if err != nil {
		t.Fatalf("first Refresh: %v", err)
	}

	_, replay := f.h.Refresh(context.Background(),
		connect.NewRequest(&authv1.RefreshRequest{RefreshToken: first.GetRefreshToken()}))
	if connect.CodeOf(replay) != connect.CodeUnauthenticated {
		t.Fatalf("replay code = %v, want unauthenticated", connect.CodeOf(replay))
	}

	_, victim := f.h.Refresh(context.Background(),
		connect.NewRequest(&authv1.RefreshRequest{RefreshToken: rotated.Msg.GetRefreshToken()}))
	if connect.CodeOf(victim) != connect.CodeUnauthenticated {
		t.Fatalf("the chain was not revoked: victim refresh returned %v", connect.CodeOf(victim))
	}
}

func TestRefreshRejectsUnknownExpiredAndRevokedTokens(t *testing.T) {
	f := newFixture(t)
	f.register(t, "ada@example.test", "correct horse")

	_, err := f.h.Refresh(context.Background(),
		connect.NewRequest(&authv1.RefreshRequest{RefreshToken: "never-issued"}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Errorf("unknown token: code = %v", connect.CodeOf(err))
	}

	_, err = f.h.Refresh(context.Background(),
		connect.NewRequest(&authv1.RefreshRequest{RefreshToken: "   "}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("empty token: code = %v, want invalid_argument", connect.CodeOf(err))
	}

	f.h.now = func() time.Time { return time.Now().Add(-365 * 24 * time.Hour) }
	stale := f.login(t, "ada@example.test", "correct horse")
	f.h.now = time.Now

	_, err = f.h.Refresh(context.Background(),
		connect.NewRequest(&authv1.RefreshRequest{RefreshToken: stale.GetRefreshToken()}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Errorf("expired token: code = %v", connect.CodeOf(err))
	}
}

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

func TestLogoutWithAnUnknownTokenSucceeds(t *testing.T) {
	f := newFixture(t)
	if _, err := f.h.Logout(context.Background(),
		connect.NewRequest(&authv1.LogoutRequest{RefreshToken: "never-issued"})); err != nil {
		t.Fatalf("Logout: %v", err)
	}
}

func TestStoreFailureBecomesInternal(t *testing.T) {
	f := newFixture(t)
	f.store.failOn["CreateUser"] = errBoom

	_, err := f.h.Register(context.Background(), connect.NewRequest(&authv1.RegisterRequest{
		Email: "ada@example.test", Password: "correct horse",
	}))
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("code = %v, want internal", connect.CodeOf(err))
	}
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

func keysOf(m map[string]db.User) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func pgUUID(u pgtype.UUID) string { return pgconv.UUIDString(u) }

func TestRefreshCannotResurrectARevokedChain(t *testing.T) {
	f := newFixture(t)
	f.register(t, "ada@example.test", "correct horse")
	session := f.login(t, "ada@example.test", "correct horse")

	rotated, err := f.h.Refresh(context.Background(),
		connect.NewRequest(&authv1.RefreshRequest{RefreshToken: session.GetRefreshToken()}))
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}

	newest := hashToken(rotated.Msg.GetRefreshToken())
	row := f.store.tokens[newest]

	if _, err := f.store.RevokeChain(context.Background(), row.ChainID); err != nil {
		t.Fatalf("RevokeChain: %v", err)
	}
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

func TestLoginRejectsAnOversizePasswordWithoutALookup(t *testing.T) {
	f := newFixture(t)
	f.store.failOn["GetUserByEmail"] = errBoom

	_, err := f.h.Login(context.Background(), connect.NewRequest(&authv1.LoginRequest{
		Email:    "ada@example.test",
		Password: strings.Repeat("x", maxPasswordBytes+1),
	}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("code = %v, want unauthenticated (err=%v)", connect.CodeOf(err), err)
	}
}

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

	_, err = f.h.Login(context.Background(), connect.NewRequest(&authv1.LoginRequest{
		Email: "ada@example.test", Password: "correct horse",
	}))
	if connect.CodeOf(err) != connect.CodeResourceExhausted {
		t.Fatalf("code = %v, want resource_exhausted while locked", connect.CodeOf(err))
	}
}

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

	for i := 0; i < 2; i++ {
		_, err := f.h.Login(context.Background(), connect.NewRequest(&authv1.LoginRequest{
			Email: "ada@example.test", Password: "wrong",
		}))
		if connect.CodeOf(err) != connect.CodeUnauthenticated {
			t.Fatalf("attempt %d code = %v, want unauthenticated", i+1, connect.CodeOf(err))
		}
	}
}

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
