package session

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strconv"
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
	links   map[string]db.TelegramLink
	deleted []string
	offsets map[string]int64
}

func newFakeQueries() *fakeQueries {
	return &fakeQueries{links: map[string]db.TelegramLink{}, offsets: map[string]int64{}}
}

func key(bot string, id int64) string { return bot + ":" + strconv.FormatInt(id, 10) }

func (f *fakeQueries) UpsertLink(_ context.Context, arg db.UpsertLinkParams) (db.TelegramLink, error) {
	link := db.TelegramLink{
		Bot:              arg.Bot,
		TelegramUserID:   arg.TelegramUserID,
		UserID:           arg.UserID,
		TelegramUsername: arg.TelegramUsername,
		ChatID:           arg.ChatID,
		AccessToken:      arg.AccessToken,
		AccessExpiresAt:  arg.AccessExpiresAt,
		RefreshToken:     arg.RefreshToken,
	}
	f.links[key(arg.Bot, arg.TelegramUserID)] = link
	return link, nil
}

func (f *fakeQueries) GetLink(_ context.Context, arg db.GetLinkParams) (db.TelegramLink, error) {
	link, ok := f.links[key(arg.Bot, arg.TelegramUserID)]
	if !ok {
		return db.TelegramLink{}, pgx.ErrNoRows
	}
	return link, nil
}

func (f *fakeQueries) UpdateLinkTokens(
	_ context.Context, arg db.UpdateLinkTokensParams,
) (db.TelegramLink, error) {
	link, ok := f.links[key(arg.Bot, arg.TelegramUserID)]
	if !ok {
		return db.TelegramLink{}, pgx.ErrNoRows
	}
	link.AccessToken = arg.AccessToken
	link.AccessExpiresAt = arg.AccessExpiresAt
	link.RefreshToken = arg.RefreshToken
	f.links[key(arg.Bot, arg.TelegramUserID)] = link
	return link, nil
}

func (f *fakeQueries) DeleteLink(_ context.Context, arg db.DeleteLinkParams) (int64, error) {
	k := key(arg.Bot, arg.TelegramUserID)
	if _, ok := f.links[k]; !ok {
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
	accessToken string
	redeemErr   error
	refreshErr  error

	redeemed   []*authv1.RedeemLinkTokenRequest
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
	if f.refreshErr != nil {
		return nil, f.refreshErr
	}
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
	_ context.Context, arg db.ExpireLinkAccessParams,
) (int64, error) {
	k := key(arg.Bot, arg.TelegramUserID)
	link, ok := f.links[k]
	if !ok {
		return 0, nil
	}
	link.AccessExpiresAt = pgtype.Timestamptz{Time: time.Now().Add(-time.Hour), Valid: true}
	f.links[k] = link
	return 1, nil
}
