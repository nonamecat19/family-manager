package handler

import (
	"context"
	"log/slog"
	"net/http"
	"net/netip"
	"regexp"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jackc/pgx/v5/pgtype"

	fmauth "github.com/nnc/family-manager/libs/go/auth"
	"github.com/nnc/family-manager/libs/go/database/pgconv"
	authv1 "github.com/nnc/family-manager/sdk/go/auth/v1"
	"github.com/nnc/family-manager/services/auth/db"
	"github.com/nnc/family-manager/services/auth/internal/ratelimit"
	"github.com/nnc/family-manager/services/auth/internal/throttle"
	"github.com/nnc/family-manager/services/auth/internal/token"
)

type clock struct{ t time.Time }

func (c *clock) now() time.Time          { return c.t }
func (c *clock) advance(d time.Duration) { c.t = c.t.Add(d) }
func withUser(userID string) context.Context {
	return fmauth.WithClaims(context.Background(), &fmauth.Claims{UserID: userID})
}

func (f *fixture) withClock() *clock {
	c := &clock{t: time.Now()}
	f.h.now = c.now
	return c
}

func (f *fixture) startLogin(t *testing.T, kind authv1.DeviceLoginKind) *authv1.StartDeviceLoginResponse {
	t.Helper()
	res, err := f.h.StartDeviceLogin(context.Background(),
		connect.NewRequest(&authv1.StartDeviceLoginRequest{Kind: kind}))
	if err != nil {
		t.Fatalf("StartDeviceLogin: %v", err)
	}
	return res.Msg
}

func (f *fixture) poll(t *testing.T, deviceCode string) *authv1.PollDeviceLoginResponse {
	t.Helper()
	res, err := f.h.PollDeviceLogin(context.Background(),
		connect.NewRequest(&authv1.PollDeviceLoginRequest{DeviceCode: deviceCode}))
	if err != nil {
		t.Fatalf("PollDeviceLogin: %v", err)
	}
	return res.Msg
}

func (f *fixture) sessionOf(userID string) (string, pgtype.UUID) {
	id, err := pgconv.UUID(userID)
	if err != nil {
		panic(err)
	}
	user, err := f.store.GetUserByID(context.Background(), id)
	if err != nil {
		panic(err)
	}
	chainID := mustChainID()
	tokens, err := f.h.mintSession(context.Background(), user, chainID, true)
	if err != nil {
		panic(err)
	}
	return tokens.access, chainID
}

func approveRequest(access, userCode string) *connect.Request[authv1.ApproveDeviceLoginRequest] {
	req := connect.NewRequest(&authv1.ApproveDeviceLoginRequest{UserCode: userCode})
	req.Header().Set("Authorization", "Bearer "+access)
	return req
}

func (f *fixture) approveAs(access, userID, userCode string) error {
	_, err := f.h.ApproveDeviceLogin(withUser(userID), approveRequest(access, userCode))
	return err
}

func (f *fixture) approve(userID, userCode string) error {
	access, _ := f.sessionOf(userID)
	return f.approveAs(access, userID, userCode)
}

var userCodeShape = regexp.MustCompile(`^[BCDFGHJKLMNPQRSTVWXZ]{4}-[BCDFGHJKLMNPQRSTVWXZ]{4}$`)

func TestStartDeviceLoginIssuesCodes(t *testing.T) {
	f := newFixture(t)
	c := f.withClock()

	res := f.startLogin(t, authv1.DeviceLoginKind_DEVICE_LOGIN_KIND_UNSPECIFIED)

	if !userCodeShape.MatchString(res.GetUserCode()) {
		t.Fatalf("user code = %q, want XXXX-XXXX from the unambiguous alphabet", res.GetUserCode())
	}
	if len(res.GetDeviceCode()) < 40 {
		t.Fatalf("device code %q is too short to be a secret", res.GetDeviceCode())
	}
	if res.GetIntervalSeconds() != 5 {
		t.Fatalf("interval = %d, want 5", res.GetIntervalSeconds())
	}
	if got := res.GetExpiresAt().AsTime(); !got.Equal(c.t.Add(deviceLoginTTL)) {
		t.Fatalf("expires_at = %v, want %v", got, c.t.Add(deviceLoginTTL))
	}
	if res.GetTelegramStartPayload() != "" {
		t.Fatalf("a device grant carried a telegram payload %q", res.GetTelegramStartPayload())
	}

	for _, g := range f.store.grants {
		if g.Kind != "device" {
			t.Fatalf("kind = %q, want device", g.Kind)
		}
		plain := strings.ReplaceAll(res.GetUserCode(), "-", "")
		if g.DeviceCodeHash == res.GetDeviceCode() || g.UserCodeHash == plain {
			t.Fatal("a login grant code was stored in plain text")
		}
	}
}

func TestStartTelegramLoginCarriesAStartPayload(t *testing.T) {
	f := newFixture(t)

	res := f.startLogin(t, authv1.DeviceLoginKind_DEVICE_LOGIN_KIND_TELEGRAM)

	want := "login_" + strings.ReplaceAll(res.GetUserCode(), "-", "")
	if res.GetTelegramStartPayload() != want {
		t.Fatalf("payload = %q, want %q", res.GetTelegramStartPayload(), want)
	}
	if !regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`).MatchString(want) {
		t.Fatalf("payload %q does not fit a Telegram start parameter", want)
	}
}

func TestStartDeviceLoginRejectsUnknownKind(t *testing.T) {
	f := newFixture(t)

	_, err := f.h.StartDeviceLogin(context.Background(),
		connect.NewRequest(&authv1.StartDeviceLoginRequest{Kind: 99}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want invalid_argument", connect.CodeOf(err))
	}
}

func TestStartDeviceLoginRetriesAUserCodeCollision(t *testing.T) {
	f := newFixture(t)
	f.store.failOn["CreateLoginGrant"] = uniqueViolation{}

	_, err := f.h.StartDeviceLogin(context.Background(),
		connect.NewRequest(&authv1.StartDeviceLoginRequest{}))
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("code = %v, want internal after every attempt collided", connect.CodeOf(err))
	}
}

func TestPollDeviceLoginIsPendingThenSlowsDown(t *testing.T) {
	f := newFixture(t)
	c := f.withClock()
	start := f.startLogin(t, authv1.DeviceLoginKind_DEVICE_LOGIN_KIND_DEVICE)

	if got := f.poll(t, start.GetDeviceCode()).GetStatus(); got != authv1.DeviceLoginStatus_DEVICE_LOGIN_STATUS_PENDING {
		t.Fatalf("first poll = %v, want pending", got)
	}

	c.advance(time.Second)
	fast := f.poll(t, start.GetDeviceCode())
	if fast.GetStatus() != authv1.DeviceLoginStatus_DEVICE_LOGIN_STATUS_SLOW_DOWN {
		t.Fatalf("fast poll = %v, want slow_down", fast.GetStatus())
	}
	if fast.GetIntervalSeconds() <= start.GetIntervalSeconds() {
		t.Fatalf("slow_down interval = %d, want more than %d",
			fast.GetIntervalSeconds(), start.GetIntervalSeconds())
	}

	c.advance(deviceLoginInterval + deviceLoginSlowDown)
	if got := f.poll(t, start.GetDeviceCode()).GetStatus(); got != authv1.DeviceLoginStatus_DEVICE_LOGIN_STATUS_PENDING {
		t.Fatalf("poll after waiting = %v, want pending", got)
	}
}

func TestPollDeviceLoginToleratesJitter(t *testing.T) {
	f := newFixture(t)
	c := f.withClock()
	start := f.startLogin(t, authv1.DeviceLoginKind_DEVICE_LOGIN_KIND_DEVICE)

	f.poll(t, start.GetDeviceCode())
	c.advance(deviceLoginInterval - 500*time.Millisecond)
	if got := f.poll(t, start.GetDeviceCode()).GetStatus(); got != authv1.DeviceLoginStatus_DEVICE_LOGIN_STATUS_PENDING {
		t.Fatalf("poll half a second early = %v, want pending", got)
	}
}

func TestApprovedDeviceLoginMintsASessionOnce(t *testing.T) {
	f := newFixture(t)
	userID := f.register(t, "ada@example.test", "correct horse")
	start := f.startLogin(t, authv1.DeviceLoginKind_DEVICE_LOGIN_KIND_DEVICE)

	if err := f.approve(userID, start.GetUserCode()); err != nil {
		t.Fatalf("ApproveDeviceLogin: %v", err)
	}
	chainsBefore := len(f.store.chains)

	res := f.poll(t, start.GetDeviceCode())
	if res.GetStatus() != authv1.DeviceLoginStatus_DEVICE_LOGIN_STATUS_APPROVED {
		t.Fatalf("status = %v, want approved", res.GetStatus())
	}
	if res.GetAccessToken() == "" || res.GetRefreshToken() == "" || res.GetExpiresIn() <= 0 {
		t.Fatal("approved poll returned an incomplete token pair")
	}
	if f.signer.last().UserID != userID {
		t.Fatalf("token minted for %q, want %q", f.signer.last().UserID, userID)
	}
	if len(f.store.chains) != chainsBefore+1 {
		t.Fatalf("chains = %d, want a fresh chain (%d)", len(f.store.chains), chainsBefore+1)
	}

	if _, err := f.h.Refresh(context.Background(),
		connect.NewRequest(&authv1.RefreshRequest{RefreshToken: res.GetRefreshToken()})); err != nil {
		t.Fatalf("the device session does not refresh: %v", err)
	}

	again := f.poll(t, start.GetDeviceCode())
	if again.GetStatus() != authv1.DeviceLoginStatus_DEVICE_LOGIN_STATUS_EXPIRED || again.GetAccessToken() != "" {
		t.Fatalf("second poll = %v with a token %t, want expired and empty",
			again.GetStatus(), again.GetAccessToken() != "")
	}
}

func TestApproveAcceptsTheCodeAsTyped(t *testing.T) {
	f := newFixture(t)
	userID := f.register(t, "ada@example.test", "correct horse")
	start := f.startLogin(t, authv1.DeviceLoginKind_DEVICE_LOGIN_KIND_DEVICE)

	typed := " " + strings.ToLower(strings.ReplaceAll(start.GetUserCode(), "-", " ")) + " "
	if err := f.approve(userID, typed); err != nil {
		t.Fatalf("ApproveDeviceLogin(%q): %v", typed, err)
	}
}

func TestApproveDeviceLoginNeedsClaims(t *testing.T) {
	f := newFixture(t)
	start := f.startLogin(t, authv1.DeviceLoginKind_DEVICE_LOGIN_KIND_DEVICE)

	_, err := f.h.ApproveDeviceLogin(context.Background(),
		connect.NewRequest(&authv1.ApproveDeviceLoginRequest{UserCode: start.GetUserCode()}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("code = %v, want unauthenticated", connect.CodeOf(err))
	}
}

func TestApproveRejectsUnknownAndMalformedCodes(t *testing.T) {
	f := newFixture(t)
	userID := f.register(t, "ada@example.test", "correct horse")
	f.startLogin(t, authv1.DeviceLoginKind_DEVICE_LOGIN_KIND_DEVICE)

	for _, code := range []string{"", "BCDF-GHJK", "AAAA-AAAA", "BCDF-GHJKL", "BCD", "login_BCDFGHJK"} {
		if err := f.approve(userID, code); connect.CodeOf(err) != connect.CodeNotFound {
			t.Errorf("approve(%q) code = %v, want not_found", code, connect.CodeOf(err))
		}
	}
}

func TestApproveIsIdempotentForTheApproverOnly(t *testing.T) {
	f := newFixture(t)
	ada := f.register(t, "ada@example.test", "correct horse")
	eve := f.register(t, "eve@example.test", "correct horse")
	start := f.startLogin(t, authv1.DeviceLoginKind_DEVICE_LOGIN_KIND_TELEGRAM)

	if err := f.approve(ada, start.GetUserCode()); err != nil {
		t.Fatalf("first approve: %v", err)
	}
	access, _ := f.sessionOf(ada)
	res, err := f.h.ApproveDeviceLogin(withUser(ada), approveRequest(access, start.GetUserCode()))
	if err != nil {
		t.Fatalf("repeat approve by the approver: %v", err)
	}
	if res.Msg.GetKind() != authv1.DeviceLoginKind_DEVICE_LOGIN_KIND_TELEGRAM {
		t.Fatalf("kind = %v, want telegram", res.Msg.GetKind())
	}
	if err := f.approve(eve, start.GetUserCode()); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("approve by another user = %v, want not_found", connect.CodeOf(err))
	}

	f.poll(t, start.GetDeviceCode())
	if f.signer.last().UserID != ada {
		t.Fatalf("session minted for %q, want the first approver %q", f.signer.last().UserID, ada)
	}
}

func TestDeniedDeviceLoginNeverMints(t *testing.T) {
	f := newFixture(t)
	userID := f.register(t, "ada@example.test", "correct horse")
	start := f.startLogin(t, authv1.DeviceLoginKind_DEVICE_LOGIN_KIND_DEVICE)

	if _, err := f.h.DenyDeviceLogin(withUser(userID),
		connect.NewRequest(&authv1.DenyDeviceLoginRequest{UserCode: start.GetUserCode()})); err != nil {
		t.Fatalf("DenyDeviceLogin: %v", err)
	}
	if _, err := f.h.DenyDeviceLogin(withUser(userID),
		connect.NewRequest(&authv1.DenyDeviceLoginRequest{UserCode: start.GetUserCode()})); err != nil {
		t.Fatalf("repeat deny: %v", err)
	}
	if err := f.approve(userID, start.GetUserCode()); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("approve after deny = %v, want not_found", connect.CodeOf(err))
	}

	res := f.poll(t, start.GetDeviceCode())
	if res.GetStatus() != authv1.DeviceLoginStatus_DEVICE_LOGIN_STATUS_DENIED || res.GetAccessToken() != "" {
		t.Fatalf("status = %v, want denied with no token", res.GetStatus())
	}
}

func TestExpiredDeviceLoginCannotBeApprovedOrPolled(t *testing.T) {
	f := newFixture(t)
	c := f.withClock()
	userID := f.register(t, "ada@example.test", "correct horse")
	start := f.startLogin(t, authv1.DeviceLoginKind_DEVICE_LOGIN_KIND_DEVICE)

	c.advance(deviceLoginTTL)

	if err := f.approve(userID, start.GetUserCode()); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("approve after expiry = %v, want not_found", connect.CodeOf(err))
	}
	if got := f.poll(t, start.GetDeviceCode()).GetStatus(); got != authv1.DeviceLoginStatus_DEVICE_LOGIN_STATUS_EXPIRED {
		t.Fatalf("poll after expiry = %v, want expired", got)
	}
}

func TestApprovedButExpiredGrantIsNotRedeemed(t *testing.T) {
	f := newFixture(t)
	c := f.withClock()
	userID := f.register(t, "ada@example.test", "correct horse")
	start := f.startLogin(t, authv1.DeviceLoginKind_DEVICE_LOGIN_KIND_DEVICE)

	if err := f.approve(userID, start.GetUserCode()); err != nil {
		t.Fatalf("ApproveDeviceLogin: %v", err)
	}
	c.advance(deviceLoginTTL)

	res := f.poll(t, start.GetDeviceCode())
	if res.GetStatus() != authv1.DeviceLoginStatus_DEVICE_LOGIN_STATUS_EXPIRED || res.GetAccessToken() != "" {
		t.Fatalf("status = %v, want expired with no token", res.GetStatus())
	}
}

func TestPollUnknownDeviceCodeReadsAsExpired(t *testing.T) {
	f := newFixture(t)

	if got := f.poll(t, "not-a-device-code").GetStatus(); got != authv1.DeviceLoginStatus_DEVICE_LOGIN_STATUS_EXPIRED {
		t.Fatalf("status = %v, want expired", got)
	}

	_, err := f.h.PollDeviceLogin(context.Background(),
		connect.NewRequest(&authv1.PollDeviceLoginRequest{}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("empty device code = %v, want invalid_argument", connect.CodeOf(err))
	}
}

func TestApproveGuessingIsThrottled(t *testing.T) {
	f := newFixture(t)
	f.h.decisions = throttle.New(throttle.Params{Threshold: 3, Base: time.Minute, Max: time.Minute}, nil)
	userID := f.register(t, "ada@example.test", "correct horse")
	start := f.startLogin(t, authv1.DeviceLoginKind_DEVICE_LOGIN_KIND_DEVICE)

	for range 3 {
		if err := f.approve(userID, "BCDF-GHJK"); connect.CodeOf(err) != connect.CodeNotFound {
			t.Fatalf("guess = %v, want not_found", connect.CodeOf(err))
		}
	}
	if err := f.approve(userID, start.GetUserCode()); connect.CodeOf(err) != connect.CodeResourceExhausted {
		t.Fatalf("approve after guessing = %v, want resource_exhausted", connect.CodeOf(err))
	}
}

func TestApproveRecordsTheApprovingSession(t *testing.T) {
	f := newFixture(t)
	ada := f.register(t, "ada@example.test", "correct horse")
	start := f.startLogin(t, authv1.DeviceLoginKind_DEVICE_LOGIN_KIND_TELEGRAM)
	access, chainID := f.sessionOf(ada)

	if err := f.approveAs(access, ada, start.GetUserCode()); err != nil {
		t.Fatalf("approve: %v", err)
	}
	f.poll(t, start.GetDeviceCode())

	grant := f.onlyGrant(t)
	if grant.ApproverChainID != chainID {
		t.Fatalf("approver chain = %v, want %v", grant.ApproverChainID, chainID)
	}
	if !grant.ChainID.Valid || pgconv.UUIDString(grant.ChainID) != f.signer.last().ChainID {
		t.Fatalf("minted chain = %v, want the polled session's chain %q", grant.ChainID, f.signer.last().ChainID)
	}
}

func TestApproveRefusesASessionWithoutAChain(t *testing.T) {
	f := newFixture(t)
	ada := f.register(t, "ada@example.test", "correct horse")
	start := f.startLogin(t, authv1.DeviceLoginKind_DEVICE_LOGIN_KIND_DEVICE)
	legacy, err := f.signer.Sign(token.Claims{UserID: ada})
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	if err := f.approveAs(legacy, ada, start.GetUserCode()); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("approve with a chainless token = %v, want unauthenticated", connect.CodeOf(err))
	}
	if err := f.approveAs("forged", ada, start.GetUserCode()); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("approve with an unverifiable token = %v, want unauthenticated", connect.CodeOf(err))
	}
}

func TestGrantApprovedFromARevokedSessionIsNotRedeemed(t *testing.T) {
	f := newFixture(t)
	ada := f.register(t, "ada@example.test", "correct horse")
	start := f.startLogin(t, authv1.DeviceLoginKind_DEVICE_LOGIN_KIND_TELEGRAM)
	access, chainID := f.sessionOf(ada)

	if err := f.approveAs(access, ada, start.GetUserCode()); err != nil {
		t.Fatalf("approve: %v", err)
	}
	if err := revokeChain(context.Background(), f.store, chainID); err != nil {
		t.Fatalf("revoke: %v", err)
	}

	res := f.poll(t, start.GetDeviceCode())
	if res.GetStatus() != authv1.DeviceLoginStatus_DEVICE_LOGIN_STATUS_EXPIRED || res.GetAccessToken() != "" {
		t.Fatalf("poll = %v with a token %t, want expired and empty", res.GetStatus(), res.GetAccessToken() != "")
	}
}

func TestDecisionThrottleDoesNotShareLoginKeys(t *testing.T) {
	f := newFixture(t)
	f.h.throttle = throttle.New(throttle.Params{Threshold: 1, Base: time.Minute, Max: time.Minute}, nil)
	f.h.decisions = throttle.New(throttle.Params{Threshold: 1, Base: time.Minute, Max: time.Minute}, nil)
	ada := f.register(t, "ada@example.test", "correct horse")
	collidingEmail := "device-login:" + ada

	if err := f.approve(ada, "BCDF-GHJK"); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("guess = %v, want not_found", connect.CodeOf(err))
	}
	if wait := f.h.throttle.Retry(collidingEmail); wait != 0 {
		t.Fatalf("a wrong device code locked the login key for %v", wait)
	}

	f.h.decisions = throttle.New(throttle.Params{Threshold: 1, Base: time.Minute, Max: time.Minute}, nil)
	_, err := f.h.Login(context.Background(), connect.NewRequest(&authv1.LoginRequest{
		Email: collidingEmail, Password: "wrong",
	}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("login = %v, want unauthenticated", connect.CodeOf(err))
	}
	start := f.startLogin(t, authv1.DeviceLoginKind_DEVICE_LOGIN_KIND_DEVICE)
	if err := f.approve(ada, start.GetUserCode()); err != nil {
		t.Fatalf("approve after a failed login on the colliding key: %v", err)
	}
}

func TestStartDeviceLoginIsRateLimitedPerClient(t *testing.T) {
	f := newFixture(t)
	f.h.starts = ratelimit.New(2, time.Minute, nil)

	for i := range 2 {
		if _, err := f.h.StartDeviceLogin(context.Background(),
			connect.NewRequest(&authv1.StartDeviceLoginRequest{})); err != nil {
			t.Fatalf("start %d: %v", i, err)
		}
	}
	_, err := f.h.StartDeviceLogin(context.Background(), connect.NewRequest(&authv1.StartDeviceLoginRequest{}))
	if connect.CodeOf(err) != connect.CodeResourceExhausted {
		t.Fatalf("third start from the same client = %v, want resource_exhausted", connect.CodeOf(err))
	}
}

func TestStartDeviceLoginIsRateLimitedPerNetwork(t *testing.T) {
	f := newFixture(t)
	f.h.networks = ratelimit.New(2, time.Minute, nil)

	for i := range 2 {
		if _, err := f.h.StartDeviceLogin(context.Background(),
			connect.NewRequest(&authv1.StartDeviceLoginRequest{})); err != nil {
			t.Fatalf("start %d: %v", i, err)
		}
	}
	_, err := f.h.StartDeviceLogin(context.Background(), connect.NewRequest(&authv1.StartDeviceLoginRequest{}))
	if connect.CodeOf(err) != connect.CodeResourceExhausted {
		t.Fatalf("third start from the same network = %v, want resource_exhausted", connect.CodeOf(err))
	}
}

func TestPendingPressureRefusesOnlyACrowdedNetwork(t *testing.T) {
	f := newFixture(t)
	f.h.maxPending = 10

	for i := range crowdedNetwork {
		if _, err := f.h.StartDeviceLogin(context.Background(),
			connect.NewRequest(&authv1.StartDeviceLoginRequest{})); err != nil {
			t.Fatalf("start %d: %v", i, err)
		}
	}
	_, err := f.h.StartDeviceLogin(context.Background(), connect.NewRequest(&authv1.StartDeviceLoginRequest{}))
	if connect.CodeOf(err) != connect.CodeResourceExhausted {
		t.Fatalf("start from a crowded network under pressure = %v, want resource_exhausted", connect.CodeOf(err))
	}

	for range 50 {
		f.h.networks = ratelimit.New(0, time.Minute, nil)
		f.startLogin(t, authv1.DeviceLoginKind_DEVICE_LOGIN_KIND_DEVICE)
	}
	if n := len(f.store.grants); n <= int(f.h.maxPending) {
		t.Fatalf("grants = %d, want the ceiling exceeded to prove nobody else is refused", n)
	}
	f.h.networks.Take("203.0.113.0/24")
	if err := f.h.checkPendingPressure(context.Background(), "203.0.113.0/24"); err != nil {
		t.Fatalf("a quiet network was refused over the ceiling: %v", err)
	}
}

func TestPendingPressureLeavesBusyNetworksAloneBelowEightyPercent(t *testing.T) {
	f := newFixture(t)
	f.h.maxPending = 1000

	for i := range crowdedNetwork + 5 {
		if _, err := f.h.StartDeviceLogin(context.Background(),
			connect.NewRequest(&authv1.StartDeviceLoginRequest{})); err != nil {
			t.Fatalf("start %d without pressure: %v", i, err)
		}
	}
}

func TestPendingCeilingWarnsAtEightyPercent(t *testing.T) {
	f := newFixture(t)
	var logs strings.Builder
	f.h.log = slog.New(slog.NewTextHandler(&logs, nil))
	f.h.maxPending = 5

	for range 4 {
		f.startLogin(t, authv1.DeviceLoginKind_DEVICE_LOGIN_KIND_DEVICE)
	}
	if strings.Contains(logs.String(), "near the ceiling") {
		t.Fatal("warned below 80%")
	}
	f.startLogin(t, authv1.DeviceLoginKind_DEVICE_LOGIN_KIND_DEVICE)
	if strings.Count(logs.String(), "near the ceiling") != 1 {
		t.Fatalf("logs = %q, want one warning at 80%%", logs.String())
	}
}

func TestClientIPTakesOnlyTheRightmostHopFromATrustedPeer(t *testing.T) {
	proxies := TrustPeers(false, []netip.Prefix{netip.MustParsePrefix("172.18.0.0/16")})
	everyone := TrustPeers(true, nil)
	cases := []struct {
		name, forwarded, peer string
		trust                 func(netip.Addr) bool
		want                  string
	}{
		{"trusted peer", "203.0.113.7", "172.18.0.5:41234", proxies, "203.0.113.7"},
		{"client-supplied hop before caddy's", "6.6.6.6, 203.0.113.7", "172.18.0.5:41234", proxies, "203.0.113.7"},
		{"no walking past a private hop", "203.0.113.7, 10.0.0.9", "172.18.0.5:41234", proxies, "10.0.0.9"},
		{"untrusted peer is the client", "6.6.6.6", "198.51.100.4:5000", proxies, "198.51.100.4"},
		{"no trust configured", "6.6.6.6", "172.18.0.5:41234", nil, "172.18.0.5"},
		{"no header", "", "172.18.0.5:41234", proxies, "172.18.0.5"},
		{"garbage falls back to the peer", "not-an-ip", "172.18.0.5:41234", proxies, "172.18.0.5"},
		{"trust every peer", "203.0.113.7", "172.20.0.9:80", everyone, "203.0.113.7"},
		{"ipv6 client", "2001:db8::1", "[::ffff:172.18.0.5]:80", proxies, "2001:db8::1"},
	}
	for _, c := range cases {
		h := http.Header{}
		if c.forwarded != "" {
			h.Add("X-Forwarded-For", c.forwarded)
		}
		if got := clientIP(h, c.peer, c.trust); got.String() != c.want {
			t.Errorf("%s: clientIP = %v, want %s", c.name, got, c.want)
		}
	}

	h := http.Header{}
	h.Add("X-Forwarded-For", "6.6.6.6")
	h.Add("X-Forwarded-For", "203.0.113.7")
	if got := clientIP(h, "172.18.0.5:1", proxies); got.String() != "203.0.113.7" {
		t.Errorf("repeated headers: clientIP = %v, want the last one", got)
	}
}

func TestLimitKeysGroupClientsAndNetworks(t *testing.T) {
	a := netip.MustParseAddr("2001:db8:1:2::1")
	b := netip.MustParseAddr("2001:db8:1:2:ffff:ffff:ffff:ffff")
	c := netip.MustParseAddr("2001:db8:1:3::1")
	d := netip.MustParseAddr("2001:db8:2:3::1")
	if startLimitKey(a) != startLimitKey(b) || startLimitKey(a) == startLimitKey(c) {
		t.Fatal("client keys do not group by /64")
	}
	if networkKey(a) != networkKey(c) || networkKey(a) == networkKey(d) {
		t.Fatal("network keys do not group by /48")
	}
	v4a, v4b, v4c := netip.MustParseAddr("203.0.113.7"), netip.MustParseAddr("203.0.113.8"), netip.MustParseAddr("203.0.114.7")
	if startLimitKey(v4a) == startLimitKey(v4b) {
		t.Fatal("neighbouring ipv4 clients share a key")
	}
	if networkKey(v4a) != networkKey(v4b) || networkKey(v4a) == networkKey(v4c) {
		t.Fatal("ipv4 network keys do not group by /24")
	}
}

func (f *fixture) onlyGrant(t *testing.T) db.LoginGrant {
	t.Helper()
	if len(f.store.grants) != 1 {
		t.Fatalf("grants = %d, want 1", len(f.store.grants))
	}
	for _, g := range f.store.grants {
		return g
	}
	return db.LoginGrant{}
}

func TestApproveRefusesARevokedSession(t *testing.T) {
	f := newFixture(t)
	ada := f.register(t, "ada@example.test", "correct horse")
	start := f.startLogin(t, authv1.DeviceLoginKind_DEVICE_LOGIN_KIND_DEVICE)
	access, chainID := f.sessionOf(ada)
	if err := revokeChain(context.Background(), f.store, chainID); err != nil {
		t.Fatalf("revoke: %v", err)
	}

	if err := f.approveAs(access, ada, start.GetUserCode()); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("approve from a revoked session = %v, want unauthenticated", connect.CodeOf(err))
	}
	if f.onlyGrant(t).ApprovedAt.Valid {
		t.Fatal("a revoked session approved the grant")
	}
}
