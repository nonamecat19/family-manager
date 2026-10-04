package session

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	authv1 "github.com/nnc/family-manager/sdk/go/auth/v1"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nnc/family-manager/apps/tui/internal/credentials"
)

type fakeAuth struct {
	start       *authv1.StartDeviceLoginResponse
	startKind   authv1.DeviceLoginKind
	polls       []*authv1.PollDeviceLoginResponse
	pollCodes   []string
	refresh     func(*authv1.RefreshRequest) (*authv1.RefreshResponse, error)
	refreshes   int
	logoutToken string
}

func (f *fakeAuth) StartDeviceLogin(_ context.Context, req *connect.Request[authv1.StartDeviceLoginRequest]) (*connect.Response[authv1.StartDeviceLoginResponse], error) {
	f.startKind = req.Msg.GetKind()
	return connect.NewResponse(f.start), nil
}

func (f *fakeAuth) PollDeviceLogin(_ context.Context, req *connect.Request[authv1.PollDeviceLoginRequest]) (*connect.Response[authv1.PollDeviceLoginResponse], error) {
	f.pollCodes = append(f.pollCodes, req.Msg.GetDeviceCode())
	if len(f.polls) == 0 {
		return nil, connect.NewError(connect.CodeInternal, errors.New("no scripted poll left"))
	}
	next := f.polls[0]
	f.polls = f.polls[1:]
	return connect.NewResponse(next), nil
}

func (f *fakeAuth) Refresh(_ context.Context, req *connect.Request[authv1.RefreshRequest]) (*connect.Response[authv1.RefreshResponse], error) {
	f.refreshes++
	resp, err := f.refresh(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (f *fakeAuth) Logout(_ context.Context, req *connect.Request[authv1.LogoutRequest]) (*connect.Response[authv1.LogoutResponse], error) {
	f.logoutToken = req.Msg.GetRefreshToken()
	return connect.NewResponse(&authv1.LogoutResponse{}), nil
}

type memStore struct {
	c     *credentials.Credentials
	saves int
}

func (m *memStore) Load() (credentials.Credentials, error) {
	if m.c == nil {
		return credentials.Credentials{}, credentials.ErrNotFound
	}
	return *m.c, nil
}

func (m *memStore) Save(c credentials.Credentials) error {
	m.saves++
	m.c = &c
	return nil
}

func (m *memStore) Delete() error {
	m.c = nil
	return nil
}

type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

func newTestSession(auth *fakeAuth, store *memStore) (*Session, *clock) {
	clk := &clock{t: time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)}
	s := New(auth, store)
	s.now = clk.now
	return s, clk
}

func recordingSleep(clk *clock, slept *[]time.Duration) func(context.Context, time.Duration) error {
	return func(_ context.Context, d time.Duration) error {
		*slept = append(*slept, d)
		clk.t = clk.t.Add(d)
		return nil
	}
}

func status(s authv1.DeviceLoginStatus) *authv1.PollDeviceLoginResponse {
	return &authv1.PollDeviceLoginResponse{Status: s}
}

func startResponse(clk time.Time) *authv1.StartDeviceLoginResponse {
	return &authv1.StartDeviceLoginResponse{
		DeviceCode:      "device-secret",
		UserCode:        "BCDF-GHJK",
		ExpiresAt:       timestamppb.New(clk.Add(10 * time.Minute)),
		IntervalSeconds: 5,
	}
}

func token(t *testing.T, sub, family string, exp time.Time) string {
	t.Helper()
	enc := base64.RawURLEncoding
	payload := fmt.Sprintf(`{"sub":%q,"family_id":%q,"exp":%d}`, sub, family, exp.Unix())
	return enc.EncodeToString([]byte(`{"alg":"ES256"}`)) + "." + enc.EncodeToString([]byte(payload)) + ".sig"
}

func TestStartDeviceLoginAsksForDeviceKind(t *testing.T) {
	auth := &fakeAuth{}
	s, clk := newTestSession(auth, &memStore{})
	auth.start = startResponse(clk.t)
	g, err := s.StartDeviceLogin(context.Background())
	if err != nil {
		t.Fatalf("StartDeviceLogin: %v", err)
	}
	if auth.startKind != authv1.DeviceLoginKind_DEVICE_LOGIN_KIND_DEVICE {
		t.Fatalf("kind = %v, want DEVICE", auth.startKind)
	}
	if g.UserCode != "BCDF-GHJK" || g.DeviceCode != "device-secret" || g.Interval != 5*time.Second {
		t.Fatalf("grant = %+v", g)
	}
	if !g.ExpiresAt.Equal(clk.t.Add(10 * time.Minute)) {
		t.Fatalf("ExpiresAt = %v", g.ExpiresAt)
	}
}

func TestStartDeviceLoginDefaultsInterval(t *testing.T) {
	auth := &fakeAuth{start: &authv1.StartDeviceLoginResponse{DeviceCode: "d", UserCode: "u"}}
	s, _ := newTestSession(auth, &memStore{})
	g, err := s.StartDeviceLogin(context.Background())
	if err != nil {
		t.Fatalf("StartDeviceLogin: %v", err)
	}
	if g.Interval != defaultInterval {
		t.Fatalf("Interval = %v, want %v", g.Interval, defaultInterval)
	}
}

func TestWaitForApprovalStoresTokens(t *testing.T) {
	store := &memStore{}
	auth := &fakeAuth{}
	s, clk := newTestSession(auth, store)
	auth.start = startResponse(clk.t)
	access := token(t, "user-1", "family-1", clk.t.Add(15*time.Minute))
	auth.polls = []*authv1.PollDeviceLoginResponse{
		status(authv1.DeviceLoginStatus_DEVICE_LOGIN_STATUS_PENDING),
		status(authv1.DeviceLoginStatus_DEVICE_LOGIN_STATUS_PENDING),
		{Status: authv1.DeviceLoginStatus_DEVICE_LOGIN_STATUS_APPROVED, AccessToken: access, RefreshToken: "refresh-1", ExpiresIn: 900},
	}
	g, err := s.StartDeviceLogin(context.Background())
	if err != nil {
		t.Fatalf("StartDeviceLogin: %v", err)
	}
	var slept []time.Duration
	if err := s.WaitForApproval(context.Background(), &g, recordingSleep(clk, &slept)); err != nil {
		t.Fatalf("WaitForApproval: %v", err)
	}
	if len(slept) != 3 {
		t.Fatalf("slept %d times, want 3", len(slept))
	}
	for _, d := range slept {
		if d != 5*time.Second {
			t.Fatalf("slept %v, want 5s each", slept)
		}
	}
	for _, c := range auth.pollCodes {
		if c != "device-secret" {
			t.Fatalf("polled with %q", c)
		}
	}
	if store.c == nil || store.c.AccessToken != access || store.c.RefreshToken != "refresh-1" {
		t.Fatalf("stored = %+v", store.c)
	}
	if want := clk.t.Add(900 * time.Second); !store.c.ExpiresAt.Equal(want) {
		t.Fatalf("ExpiresAt = %v, want %v", store.c.ExpiresAt, want)
	}
}

func TestWaitForApprovalSlowsDown(t *testing.T) {
	store := &memStore{}
	auth := &fakeAuth{}
	s, clk := newTestSession(auth, store)
	auth.start = startResponse(clk.t)
	auth.polls = []*authv1.PollDeviceLoginResponse{
		{Status: authv1.DeviceLoginStatus_DEVICE_LOGIN_STATUS_SLOW_DOWN, IntervalSeconds: 7},
		{Status: authv1.DeviceLoginStatus_DEVICE_LOGIN_STATUS_SLOW_DOWN, IntervalSeconds: 30},
		status(authv1.DeviceLoginStatus_DEVICE_LOGIN_STATUS_PENDING),
		{Status: authv1.DeviceLoginStatus_DEVICE_LOGIN_STATUS_APPROVED, AccessToken: "a", RefreshToken: "r", ExpiresIn: 900},
	}
	g, err := s.StartDeviceLogin(context.Background())
	if err != nil {
		t.Fatalf("StartDeviceLogin: %v", err)
	}
	var slept []time.Duration
	if err := s.WaitForApproval(context.Background(), &g, recordingSleep(clk, &slept)); err != nil {
		t.Fatalf("WaitForApproval: %v", err)
	}
	want := []time.Duration{5 * time.Second, 10 * time.Second, 30 * time.Second, 30 * time.Second}
	if fmt.Sprint(slept) != fmt.Sprint(want) {
		t.Fatalf("slept %v, want %v", slept, want)
	}
}

func TestWaitForApprovalDenied(t *testing.T) {
	store := &memStore{}
	auth := &fakeAuth{}
	s, clk := newTestSession(auth, store)
	auth.start = startResponse(clk.t)
	auth.polls = []*authv1.PollDeviceLoginResponse{
		status(authv1.DeviceLoginStatus_DEVICE_LOGIN_STATUS_PENDING),
		status(authv1.DeviceLoginStatus_DEVICE_LOGIN_STATUS_DENIED),
	}
	g, err := s.StartDeviceLogin(context.Background())
	if err != nil {
		t.Fatalf("StartDeviceLogin: %v", err)
	}
	var slept []time.Duration
	err = s.WaitForApproval(context.Background(), &g, recordingSleep(clk, &slept))
	if !errors.Is(err, ErrDenied) {
		t.Fatalf("err = %v, want ErrDenied", err)
	}
	if store.c != nil {
		t.Fatalf("stored credentials after denial: %+v", store.c)
	}
}

func TestWaitForApprovalExpiredByServer(t *testing.T) {
	store := &memStore{}
	auth := &fakeAuth{}
	s, clk := newTestSession(auth, store)
	auth.start = startResponse(clk.t)
	auth.polls = []*authv1.PollDeviceLoginResponse{
		status(authv1.DeviceLoginStatus_DEVICE_LOGIN_STATUS_EXPIRED),
	}
	g, err := s.StartDeviceLogin(context.Background())
	if err != nil {
		t.Fatalf("StartDeviceLogin: %v", err)
	}
	var slept []time.Duration
	if err := s.WaitForApproval(context.Background(), &g, recordingSleep(clk, &slept)); !errors.Is(err, ErrExpired) {
		t.Fatalf("err = %v, want ErrExpired", err)
	}
	if store.c != nil {
		t.Fatalf("stored credentials after expiry: %+v", store.c)
	}
}

func TestWaitForApprovalExpiresLocallyWithoutPolling(t *testing.T) {
	auth := &fakeAuth{}
	s, clk := newTestSession(auth, &memStore{})
	auth.start = &authv1.StartDeviceLoginResponse{
		DeviceCode:      "d",
		UserCode:        "u",
		ExpiresAt:       timestamppb.New(clk.t.Add(12 * time.Second)),
		IntervalSeconds: 5,
	}
	auth.polls = []*authv1.PollDeviceLoginResponse{
		status(authv1.DeviceLoginStatus_DEVICE_LOGIN_STATUS_PENDING),
		status(authv1.DeviceLoginStatus_DEVICE_LOGIN_STATUS_PENDING),
	}
	g, err := s.StartDeviceLogin(context.Background())
	if err != nil {
		t.Fatalf("StartDeviceLogin: %v", err)
	}
	var slept []time.Duration
	if err := s.WaitForApproval(context.Background(), &g, recordingSleep(clk, &slept)); !errors.Is(err, ErrExpired) {
		t.Fatalf("err = %v, want ErrExpired", err)
	}
	if len(auth.pollCodes) != 2 {
		t.Fatalf("polled %d times, want 2", len(auth.pollCodes))
	}
}

func TestWaitForApprovalStopsOnCancel(t *testing.T) {
	auth := &fakeAuth{}
	s, clk := newTestSession(auth, &memStore{})
	auth.start = startResponse(clk.t)
	g, err := s.StartDeviceLogin(context.Background())
	if err != nil {
		t.Fatalf("StartDeviceLogin: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := s.WaitForApproval(ctx, &g, Sleep); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if len(auth.pollCodes) != 0 {
		t.Fatalf("polled after cancel")
	}
}

func TestAccessTokenUsesStoredTokenWhileFresh(t *testing.T) {
	auth := &fakeAuth{refresh: func(*authv1.RefreshRequest) (*authv1.RefreshResponse, error) {
		return nil, errors.New("must not refresh")
	}}
	store := &memStore{}
	s, clk := newTestSession(auth, store)
	store.c = &credentials.Credentials{AccessToken: "a", RefreshToken: "r", ExpiresAt: clk.t.Add(10 * time.Minute)}
	tok, err := s.AccessToken(context.Background())
	if err != nil {
		t.Fatalf("AccessToken: %v", err)
	}
	if tok != "a" || auth.refreshes != 0 {
		t.Fatalf("tok = %q, refreshes = %d", tok, auth.refreshes)
	}
}

func TestAccessTokenRefreshesWhenExpired(t *testing.T) {
	store := &memStore{}
	var presented string
	auth := &fakeAuth{refresh: func(req *authv1.RefreshRequest) (*authv1.RefreshResponse, error) {
		presented = req.GetRefreshToken()
		return &authv1.RefreshResponse{AccessToken: "a2", RefreshToken: "r2", ExpiresIn: 900}, nil
	}}
	s, clk := newTestSession(auth, store)
	store.c = &credentials.Credentials{AccessToken: "a1", RefreshToken: "r1", ExpiresAt: clk.t.Add(-time.Minute)}
	tok, err := s.AccessToken(context.Background())
	if err != nil {
		t.Fatalf("AccessToken: %v", err)
	}
	if tok != "a2" || presented != "r1" {
		t.Fatalf("tok = %q, presented = %q", tok, presented)
	}
	if store.c.RefreshToken != "r2" || !store.c.ExpiresAt.Equal(clk.t.Add(900*time.Second)) {
		t.Fatalf("stored = %+v", store.c)
	}
	tok, err = s.AccessToken(context.Background())
	if err != nil || tok != "a2" || auth.refreshes != 1 {
		t.Fatalf("second call tok = %q err = %v refreshes = %d", tok, err, auth.refreshes)
	}
}

func TestAccessTokenRefreshesInsideSkew(t *testing.T) {
	store := &memStore{}
	auth := &fakeAuth{refresh: func(*authv1.RefreshRequest) (*authv1.RefreshResponse, error) {
		return &authv1.RefreshResponse{AccessToken: "a2", RefreshToken: "r2", ExpiresIn: 900}, nil
	}}
	s, clk := newTestSession(auth, store)
	store.c = &credentials.Credentials{AccessToken: "a1", RefreshToken: "r1", ExpiresAt: clk.t.Add(10 * time.Second)}
	if tok, err := s.AccessToken(context.Background()); err != nil || tok != "a2" {
		t.Fatalf("tok = %q err = %v", tok, err)
	}
}

func TestAccessTokenRejectedRefreshLogsOut(t *testing.T) {
	store := &memStore{}
	auth := &fakeAuth{refresh: func(*authv1.RefreshRequest) (*authv1.RefreshResponse, error) {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("revoked"))
	}}
	s, clk := newTestSession(auth, store)
	store.c = &credentials.Credentials{AccessToken: "a1", RefreshToken: "r1", ExpiresAt: clk.t.Add(-time.Minute)}
	if _, err := s.AccessToken(context.Background()); !errors.Is(err, ErrLoggedOut) {
		t.Fatalf("err = %v, want ErrLoggedOut", err)
	}
	if store.c != nil {
		t.Fatalf("credentials kept after rejected refresh")
	}
}

func TestAccessTokenTransientRefreshFailureKeepsCredentials(t *testing.T) {
	store := &memStore{}
	auth := &fakeAuth{refresh: func(*authv1.RefreshRequest) (*authv1.RefreshResponse, error) {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("down"))
	}}
	s, clk := newTestSession(auth, store)
	store.c = &credentials.Credentials{AccessToken: "a1", RefreshToken: "r1", ExpiresAt: clk.t.Add(-time.Minute)}
	_, err := s.AccessToken(context.Background())
	if err == nil || errors.Is(err, ErrLoggedOut) {
		t.Fatalf("err = %v, want a transient error", err)
	}
	if store.c == nil || store.c.RefreshToken != "r1" {
		t.Fatalf("credentials changed: %+v", store.c)
	}
}

func TestAccessTokenLoggedOut(t *testing.T) {
	s, _ := newTestSession(&fakeAuth{}, &memStore{})
	if _, err := s.AccessToken(context.Background()); !errors.Is(err, ErrLoggedOut) {
		t.Fatalf("err = %v, want ErrLoggedOut", err)
	}
}

func TestClaimsDecodeStoredToken(t *testing.T) {
	store := &memStore{}
	s, clk := newTestSession(&fakeAuth{}, store)
	exp := clk.t.Add(10 * time.Minute)
	store.c = &credentials.Credentials{AccessToken: token(t, "user-1", "family-1", exp), RefreshToken: "r", ExpiresAt: exp}
	c, err := s.Claims(context.Background())
	if err != nil {
		t.Fatalf("Claims: %v", err)
	}
	if c.UserID != "user-1" || c.FamilyID != "family-1" || !c.ExpiresAt.Equal(exp.Truncate(time.Second)) {
		t.Fatalf("claims = %+v", c)
	}
}

func TestParseClaimsRejectsMalformed(t *testing.T) {
	for _, tok := range []string{"", "a.b", "a.!!!.c", "a." + base64.RawURLEncoding.EncodeToString([]byte(`{"family_id":"f"}`)) + ".c"} {
		if _, err := ParseClaims(tok); !errors.Is(err, ErrMalformedToken) {
			t.Fatalf("ParseClaims(%q) err = %v, want ErrMalformedToken", tok, err)
		}
	}
}

func TestLogoutRevokesAndDeletes(t *testing.T) {
	store := &memStore{c: &credentials.Credentials{AccessToken: "a", RefreshToken: "r"}}
	auth := &fakeAuth{}
	s, _ := newTestSession(auth, store)
	if err := s.Logout(context.Background()); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if auth.logoutToken != "r" || store.c != nil {
		t.Fatalf("logoutToken = %q, stored = %+v", auth.logoutToken, store.c)
	}
	if err := s.Logout(context.Background()); !errors.Is(err, ErrLoggedOut) {
		t.Fatalf("second Logout err = %v, want ErrLoggedOut", err)
	}
}

func TestInterceptorSetsBearer(t *testing.T) {
	store := &memStore{}
	s, clk := newTestSession(&fakeAuth{}, store)
	store.c = &credentials.Credentials{AccessToken: "a", RefreshToken: "r", ExpiresAt: clk.t.Add(time.Hour)}
	var got string
	next := func(_ context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		got = req.Header().Get("Authorization")
		return nil, nil
	}
	if _, err := Interceptor(s)(next)(context.Background(), connect.NewRequest(&authv1.LogoutRequest{})); err != nil {
		t.Fatalf("intercept: %v", err)
	}
	if got != "Bearer a" {
		t.Fatalf("Authorization = %q", got)
	}
}
