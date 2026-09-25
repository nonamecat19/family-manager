package session

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	authv1 "github.com/nnc/family-manager/sdk/go/auth/v1"
	"github.com/nnc/family-manager/services/telegram/db"
	"github.com/nnc/family-manager/services/telegram/internal/secret"
)

const testUserID = "6f1b9d3e-0d3a-4c19-9a2f-2f9a6a3d5c11"

type fakeQueries struct {
	mu      sync.Mutex
	links   map[string]db.TelegramLink
	deleted []string
	offsets map[string]int64
}

func newFakeQueries() *fakeQueries {
	return &fakeQueries{links: map[string]db.TelegramLink{}, offsets: map[string]int64{}}
}

func key(id int64) string { return strconv.FormatInt(id, 10) }

func (f *fakeQueries) UpsertLink(_ context.Context, arg db.UpsertLinkParams) (db.TelegramLink, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	link := db.TelegramLink{
		TelegramUserID:   arg.TelegramUserID,
		UserID:           arg.UserID,
		TelegramUsername: arg.TelegramUsername,
		ChatID:           arg.ChatID,
		AccessToken:      arg.AccessToken,
		AccessExpiresAt:  arg.AccessExpiresAt,
		RefreshToken:     arg.RefreshToken,
	}
	f.links[key(arg.TelegramUserID)] = link
	return link, nil
}

func (f *fakeQueries) GetLink(_ context.Context, id int64) (db.TelegramLink, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	link, ok := f.links[key(id)]
	if !ok {
		return db.TelegramLink{}, pgx.ErrNoRows
	}
	return link, nil
}

func (f *fakeQueries) UpdateLinkTokens(
	_ context.Context, arg db.UpdateLinkTokensParams,
) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	link, ok := f.links[key(arg.TelegramUserID)]
	if !ok || !bytes.Equal(link.RefreshToken, arg.PreviousRefreshToken) {
		return 0, nil
	}
	link.AccessToken = arg.AccessToken
	link.AccessExpiresAt = arg.AccessExpiresAt
	link.RefreshToken = arg.RefreshToken
	f.links[key(arg.TelegramUserID)] = link
	return 1, nil
}

func (f *fakeQueries) DeleteLinkWithToken(_ context.Context, arg db.DeleteLinkWithTokenParams) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	k := key(arg.TelegramUserID)
	link, ok := f.links[k]
	if !ok || !bytes.Equal(link.RefreshToken, arg.RefreshToken) {
		return 0, nil
	}
	delete(f.links, k)
	f.deleted = append(f.deleted, k)
	return 1, nil
}

func (f *fakeQueries) DeleteLinkForUser(_ context.Context, arg db.DeleteLinkForUserParams) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	k := key(arg.TelegramUserID)
	link, ok := f.links[k]
	if !ok || link.UserID != arg.UserID {
		return 0, nil
	}
	delete(f.links, k)
	f.deleted = append(f.deleted, k)
	return 1, nil
}

func (f *fakeQueries) ListLinksForUser(_ context.Context, _ pgtype.UUID) ([]db.TelegramLink, error) {
	out := make([]db.TelegramLink, 0, len(f.links))
	for _, link := range f.links {
		out = append(out, link)
	}
	return out, nil
}

func (f *fakeQueries) GetBotOffset(_ context.Context, bot string) (int64, error) {
	offset, ok := f.offsets[bot]
	if !ok {
		return 0, pgx.ErrNoRows
	}
	return offset, nil
}

func (f *fakeQueries) SetBotOffset(_ context.Context, arg db.SetBotOffsetParams) error {
	f.offsets[arg.Bot] = arg.OffsetID
	return nil
}

func jwtWithClaims(payload string) string {
	return "header." + base64.RawURLEncoding.EncodeToString([]byte(payload)) + ".signature"
}

type fakeAuth struct {
	mu          sync.Mutex
	accessToken string
	redeemErr   error
	refreshErr  error

	redeemed   []*authv1.RedeemLinkTokenRequest
	unlinked   []*authv1.UnlinkRequest
	bearers    []string
	unlinkErr  error
	decideErr  error
	decideOnce error
	approved   []*authv1.ApproveDeviceLoginRequest
	denied     []*authv1.DenyDeviceLoginRequest
	onRefresh  func()
	refreshed  []string
	loggedOut  []string
	accessSeed int
}

func (f *fakeAuth) RedeemLinkToken(
	_ context.Context, req *connect.Request[authv1.RedeemLinkTokenRequest],
) (*connect.Response[authv1.RedeemLinkTokenResponse], error) {
	if f.redeemErr != nil {
		return nil, f.redeemErr
	}
	f.redeemed = append(f.redeemed, req.Msg)
	access := f.accessToken
	if access == "" {
		access = "access-0"
	}
	return connect.NewResponse(&authv1.RedeemLinkTokenResponse{
		UserId:       testUserID,
		AccessToken:  access,
		RefreshToken: "refresh-0",
		ExpiresIn:    int64((15 * time.Minute).Seconds()),
	}), nil
}

func (f *fakeAuth) Refresh(
	_ context.Context, req *connect.Request[authv1.RefreshRequest],
) (*connect.Response[authv1.RefreshResponse], error) {
	if f.onRefresh != nil {
		f.onRefresh()
	}
	if f.refreshErr != nil {
		return nil, f.refreshErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.refreshed = append(f.refreshed, req.Msg.GetRefreshToken())
	f.accessSeed++
	return connect.NewResponse(&authv1.RefreshResponse{
		AccessToken:  "access-" + strconv.Itoa(f.accessSeed),
		RefreshToken: "refresh-" + strconv.Itoa(f.accessSeed),
		ExpiresIn:    int64((15 * time.Minute).Seconds()),
	}), nil
}

func (f *fakeAuth) Logout(
	_ context.Context, req *connect.Request[authv1.LogoutRequest],
) (*connect.Response[authv1.LogoutResponse], error) {
	f.loggedOut = append(f.loggedOut, req.Msg.GetRefreshToken())
	return connect.NewResponse(&authv1.LogoutResponse{}), nil
}

func newTestBox(t *testing.T) *secret.Box {
	t.Helper()
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		t.Fatalf("read key: %v", err)
	}
	box, err := secret.NewBox(base64.StdEncoding.EncodeToString(raw))
	if err != nil {
		t.Fatalf("NewBox: %v", err)
	}
	return box
}

func (f *fakeAuth) Login(
	context.Context, *connect.Request[authv1.LoginRequest],
) (*connect.Response[authv1.LoginResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errUnused)
}

func (f *fakeAuth) Register(
	context.Context, *connect.Request[authv1.RegisterRequest],
) (*connect.Response[authv1.RegisterResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errUnused)
}

func (f *fakeAuth) CreateLinkToken(
	context.Context, *connect.Request[authv1.CreateLinkTokenRequest],
) (*connect.Response[authv1.CreateLinkTokenResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errUnused)
}

var errUnused = errors.New("the bot never calls this")

func (f *fakeQueries) UpsertChatState(
	_ context.Context, arg db.UpsertChatStateParams,
) (db.ChatState, error) {
	return db.ChatState{
		Bot: arg.Bot, TelegramUserID: arg.TelegramUserID,
		Kind: arg.Kind, Payload: arg.Payload, ExpiresAt: arg.ExpiresAt,
	}, nil
}

func (f *fakeQueries) GetChatState(
	_ context.Context, _ db.GetChatStateParams,
) (db.ChatState, error) {
	return db.ChatState{}, pgx.ErrNoRows
}

func (f *fakeQueries) ClearChatState(_ context.Context, _ db.ClearChatStateParams) (int64, error) {
	return 0, nil
}

func (f *fakeQueries) DeleteExpiredChatStates(context.Context) (int64, error) { return 0, nil }

func (f *fakeQueries) ExpireLinkAccess(
	_ context.Context, id int64,
) (int64, error) {
	k := key(id)
	link, ok := f.links[k]
	if !ok {
		return 0, nil
	}
	link.AccessExpiresAt = pgtype.Timestamptz{Time: time.Now().Add(-time.Hour), Valid: true}
	f.links[k] = link
	return 1, nil
}

func (f *fakeAuth) ListIdentities(
	context.Context, *connect.Request[authv1.ListIdentitiesRequest],
) (*connect.Response[authv1.ListIdentitiesResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errUnused)
}

func (f *fakeAuth) Unlink(
	_ context.Context, req *connect.Request[authv1.UnlinkRequest],
) (*connect.Response[authv1.UnlinkResponse], error) {
	f.unlinked = append(f.unlinked, req.Msg)
	f.bearers = append(f.bearers, req.Header().Get("Authorization"))
	if f.unlinkErr != nil {
		return nil, f.unlinkErr
	}
	return connect.NewResponse(&authv1.UnlinkResponse{}), nil
}

func (f *fakeAuth) StartDeviceLogin(
	context.Context, *connect.Request[authv1.StartDeviceLoginRequest],
) (*connect.Response[authv1.StartDeviceLoginResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errUnused)
}

func (f *fakeAuth) PollDeviceLogin(
	context.Context, *connect.Request[authv1.PollDeviceLoginRequest],
) (*connect.Response[authv1.PollDeviceLoginResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errUnused)
}

func (f *fakeAuth) ApproveDeviceLogin(
	_ context.Context, req *connect.Request[authv1.ApproveDeviceLoginRequest],
) (*connect.Response[authv1.ApproveDeviceLoginResponse], error) {
	f.bearers = append(f.bearers, req.Header().Get("Authorization"))
	if f.decideErr != nil {
		return nil, f.decideErr
	}
	if err := f.decideOnce; err != nil {
		f.decideOnce = nil
		return nil, err
	}
	f.approved = append(f.approved, req.Msg)
	return connect.NewResponse(&authv1.ApproveDeviceLoginResponse{
		Kind: authv1.DeviceLoginKind_DEVICE_LOGIN_KIND_TELEGRAM,
	}), nil
}

func (f *fakeAuth) DenyDeviceLogin(
	_ context.Context, req *connect.Request[authv1.DenyDeviceLoginRequest],
) (*connect.Response[authv1.DenyDeviceLoginResponse], error) {
	f.bearers = append(f.bearers, req.Header().Get("Authorization"))
	if f.decideErr != nil {
		return nil, f.decideErr
	}
	if err := f.decideOnce; err != nil {
		f.decideOnce = nil
		return nil, err
	}
	f.denied = append(f.denied, req.Msg)
	return connect.NewResponse(&authv1.DenyDeviceLoginResponse{}), nil
}
