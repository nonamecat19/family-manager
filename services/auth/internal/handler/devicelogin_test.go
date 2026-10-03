package handler

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	fmauth "github.com/nnc/family-manager/libs/go/auth"
	authv1 "github.com/nnc/family-manager/sdk/go/auth/v1"
	"github.com/nnc/family-manager/services/auth/internal/throttle"
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

func (f *fixture) approve(userID, userCode string) error {
	_, err := f.h.ApproveDeviceLogin(withUser(userID),
		connect.NewRequest(&authv1.ApproveDeviceLoginRequest{UserCode: userCode}))
	return err
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
	chainsBefore := len(f.store.chains)

	if err := f.approve(userID, start.GetUserCode()); err != nil {
		t.Fatalf("ApproveDeviceLogin: %v", err)
	}

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
	res, err := f.h.ApproveDeviceLogin(withUser(ada),
		connect.NewRequest(&authv1.ApproveDeviceLoginRequest{UserCode: start.GetUserCode()}))
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
	f.h.throttle = throttle.New(throttle.Params{Threshold: 3, Base: time.Minute, Max: time.Minute}, nil)
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
