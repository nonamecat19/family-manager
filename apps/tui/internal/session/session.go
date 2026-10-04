package session

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"connectrpc.com/connect"
	authv1 "github.com/nnc/family-manager/sdk/go/auth/v1"

	"github.com/nnc/family-manager/apps/tui/internal/credentials"
)

const ApproveHint = "Approve it in any family-manager app under Settings → Approve a device"

const (
	defaultInterval = 5 * time.Second
	slowDownStep    = 5 * time.Second
	refreshSkew     = 30 * time.Second
)

var (
	ErrDenied    = errors.New("session: the sign-in was denied")
	ErrExpired   = errors.New("session: the code expired before it was approved")
	ErrLoggedOut = errors.New("session: not logged in")
)

type AuthClient interface {
	StartDeviceLogin(context.Context, *connect.Request[authv1.StartDeviceLoginRequest]) (*connect.Response[authv1.StartDeviceLoginResponse], error)
	PollDeviceLogin(context.Context, *connect.Request[authv1.PollDeviceLoginRequest]) (*connect.Response[authv1.PollDeviceLoginResponse], error)
	Refresh(context.Context, *connect.Request[authv1.RefreshRequest]) (*connect.Response[authv1.RefreshResponse], error)
	Logout(context.Context, *connect.Request[authv1.LogoutRequest]) (*connect.Response[authv1.LogoutResponse], error)
}

type Store interface {
	Load() (credentials.Credentials, error)
	Save(credentials.Credentials) error
	Delete() error
}

type Grant struct {
	DeviceCode string
	UserCode   string
	ExpiresAt  time.Time
	Interval   time.Duration
}

type Session struct {
	auth  AuthClient
	store Store
	now   func() time.Time
	mu    sync.Mutex
}

func New(auth AuthClient, store Store) *Session {
	return &Session{auth: auth, store: store, now: time.Now}
}

func (s *Session) StartDeviceLogin(ctx context.Context) (Grant, error) {
	resp, err := s.auth.StartDeviceLogin(ctx, connect.NewRequest(&authv1.StartDeviceLoginRequest{
		Kind: authv1.DeviceLoginKind_DEVICE_LOGIN_KIND_DEVICE,
	}))
	if err != nil {
		return Grant{}, fmt.Errorf("session: start device login: %w", err)
	}
	g := Grant{
		DeviceCode: resp.Msg.GetDeviceCode(),
		UserCode:   resp.Msg.GetUserCode(),
		Interval:   time.Duration(resp.Msg.GetIntervalSeconds()) * time.Second,
	}
	if ts := resp.Msg.GetExpiresAt(); ts != nil {
		g.ExpiresAt = ts.AsTime()
	}
	if g.Interval <= 0 {
		g.Interval = defaultInterval
	}
	return g, nil
}

func (s *Session) Poll(ctx context.Context, g *Grant) (bool, error) {
	if !g.ExpiresAt.IsZero() && !s.now().Before(g.ExpiresAt) {
		return false, ErrExpired
	}
	resp, err := s.auth.PollDeviceLogin(ctx, connect.NewRequest(&authv1.PollDeviceLoginRequest{DeviceCode: g.DeviceCode}))
	if err != nil {
		return false, fmt.Errorf("session: poll device login: %w", err)
	}
	m := resp.Msg
	switch m.GetStatus() {
	case authv1.DeviceLoginStatus_DEVICE_LOGIN_STATUS_PENDING:
		return false, nil
	case authv1.DeviceLoginStatus_DEVICE_LOGIN_STATUS_SLOW_DOWN:
		next := time.Duration(m.GetIntervalSeconds()) * time.Second
		g.Interval = max(next, g.Interval+slowDownStep)
		return false, nil
	case authv1.DeviceLoginStatus_DEVICE_LOGIN_STATUS_APPROVED:
		if err := s.store.Save(s.credentialsFrom(m.GetAccessToken(), m.GetRefreshToken(), m.GetExpiresIn())); err != nil {
			return false, fmt.Errorf("session: store tokens: %w", err)
		}
		return true, nil
	case authv1.DeviceLoginStatus_DEVICE_LOGIN_STATUS_DENIED:
		return false, ErrDenied
	case authv1.DeviceLoginStatus_DEVICE_LOGIN_STATUS_EXPIRED:
		return false, ErrExpired
	default:
		return false, fmt.Errorf("session: poll device login: unexpected status %v", m.GetStatus())
	}
}

func (s *Session) WaitForApproval(ctx context.Context, g *Grant, sleep func(context.Context, time.Duration) error) error {
	for {
		if err := sleep(ctx, g.Interval); err != nil {
			return err
		}
		done, err := s.Poll(ctx, g)
		if err != nil {
			return err
		}
		if done {
			return nil
		}
	}
}

func Sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func (s *Session) AccessToken(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, err := s.store.Load()
	if errors.Is(err, credentials.ErrNotFound) {
		return "", ErrLoggedOut
	}
	if err != nil {
		return "", fmt.Errorf("session: load credentials: %w", err)
	}
	if c.AccessToken != "" && s.now().Add(refreshSkew).Before(c.ExpiresAt) {
		return c.AccessToken, nil
	}
	resp, err := s.auth.Refresh(ctx, connect.NewRequest(&authv1.RefreshRequest{RefreshToken: c.RefreshToken}))
	if err != nil {
		if connect.CodeOf(err) == connect.CodeUnauthenticated {
			if derr := s.store.Delete(); derr != nil {
				return "", fmt.Errorf("session: refresh rejected, clear credentials: %w", derr)
			}
			return "", fmt.Errorf("%w: refresh rejected: %w", ErrLoggedOut, err)
		}
		return "", fmt.Errorf("session: refresh: %w", err)
	}
	next := s.credentialsFrom(resp.Msg.GetAccessToken(), resp.Msg.GetRefreshToken(), resp.Msg.GetExpiresIn())
	if err := s.store.Save(next); err != nil {
		return "", fmt.Errorf("session: store refreshed tokens: %w", err)
	}
	return next.AccessToken, nil
}

func (s *Session) Claims(ctx context.Context) (Claims, error) {
	tok, err := s.AccessToken(ctx)
	if err != nil {
		return Claims{}, err
	}
	return ParseClaims(tok)
}

func (s *Session) LoggedIn() (bool, error) {
	_, err := s.store.Load()
	if errors.Is(err, credentials.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("session: load credentials: %w", err)
	}
	return true, nil
}

func (s *Session) Logout(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, err := s.store.Load()
	if errors.Is(err, credentials.ErrNotFound) {
		return ErrLoggedOut
	}
	var revokeErr error
	if err == nil {
		if _, err := s.auth.Logout(ctx, connect.NewRequest(&authv1.LogoutRequest{RefreshToken: c.RefreshToken})); err != nil {
			revokeErr = fmt.Errorf("session: revoke refresh token: %w", err)
		}
	}
	if err := s.store.Delete(); err != nil {
		return fmt.Errorf("session: delete credentials: %w", err)
	}
	return revokeErr
}

func (s *Session) credentialsFrom(access, refresh string, expiresIn int64) credentials.Credentials {
	c := credentials.Credentials{AccessToken: access, RefreshToken: refresh}
	if expiresIn > 0 {
		c.ExpiresAt = s.now().Add(time.Duration(expiresIn) * time.Second)
	} else if claims, err := ParseClaims(access); err == nil {
		c.ExpiresAt = claims.ExpiresAt
	}
	return c
}

func Interceptor(s *Session) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			tok, err := s.AccessToken(ctx)
			if err != nil {
				return nil, err
			}
			req.Header().Set("Authorization", "Bearer "+tok)
			return next(ctx, req)
		}
	}
}
