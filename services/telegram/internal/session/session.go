package session

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"

	"github.com/nnc/family-manager/libs/go/database/pgconv"
	authv1 "github.com/nnc/family-manager/sdk/go/auth/v1"
	"github.com/nnc/family-manager/sdk/go/auth/v1/authv1connect"
	"github.com/nnc/family-manager/services/telegram/db"
	"github.com/nnc/family-manager/services/telegram/internal/secret"
	"github.com/nnc/family-manager/services/telegram/internal/telegram"
)

const (
	provider     = "telegram"
	refreshSkew  = time.Minute
	usernameSize = 64
)

var (
	ErrNotLinked = errors.New("session: this telegram account is not linked")
	ErrLinkAgain = errors.New("session: the link expired; start over from the app")
)

type Session struct {
	UserID      string
	AccessToken string
	Locale      string
}

type Store struct {
	q    db.Querier
	box  *secret.Box
	auth authv1connect.AuthServiceClient
	now  func() time.Time
}

type Options struct {
	Queries db.Querier
	Box     *secret.Box
	Auth    authv1connect.AuthServiceClient
	Now     func() time.Time
}

func NewStore(opts Options) *Store {
	s := &Store{q: opts.Queries, box: opts.Box, auth: opts.Auth, now: opts.Now}
	if s.now == nil {
		s.now = time.Now
	}
	return s
}

func (s *Store) Redeem(
	ctx context.Context, bot, linkToken string, from telegram.User, chatID int64,
) (*Session, error) {
	res, err := s.auth.RedeemLinkToken(ctx, connect.NewRequest(&authv1.RedeemLinkTokenRequest{
		Token:      linkToken,
		Provider:   provider,
		ExternalId: fmt.Sprint(from.ID),
	}))
	if err != nil {
		if connect.CodeOf(err) == connect.CodeUnauthenticated {
			return nil, ErrLinkAgain
		}
		return nil, fmt.Errorf("session: redeem link token: %w", err)
	}

	userID, err := pgconv.UUID(res.Msg.GetUserId())
	if err != nil {
		return nil, fmt.Errorf("session: malformed user id: %w", err)
	}

	access, err := s.box.Seal(res.Msg.GetAccessToken())
	if err != nil {
		return nil, err
	}
	refresh, err := s.box.Seal(res.Msg.GetRefreshToken())
	if err != nil {
		return nil, err
	}

	username := from.Username
	if len(username) > usernameSize {
		username = username[:usernameSize]
	}

	if _, err := s.q.UpsertLink(ctx, db.UpsertLinkParams{
		Bot:              bot,
		TelegramUserID:   from.ID,
		UserID:           userID,
		TelegramUsername: username,
		ChatID:           chatID,
		AccessToken:      access,
		AccessExpiresAt:  pgconv.TimestampFrom(s.expiry(res.Msg.GetExpiresIn())),
		RefreshToken:     refresh,
	}); err != nil {
		return nil, fmt.Errorf("session: store link: %w", err)
	}

	return newSession(res.Msg.GetUserId(), res.Msg.GetAccessToken()), nil
}

func (s *Store) Session(ctx context.Context, bot string, telegramUserID int64) (*Session, error) {
	link, err := s.q.GetLink(ctx, db.GetLinkParams{Bot: bot, TelegramUserID: telegramUserID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotLinked
		}
		return nil, fmt.Errorf("session: get link: %w", err)
	}

	if link.AccessExpiresAt.Valid && link.AccessExpiresAt.Time.After(s.now().Add(refreshSkew)) {
		access, err := s.box.Open(link.AccessToken)
		if err != nil {
			return nil, err
		}
		return newSession(pgconv.UUIDString(link.UserID), access), nil
	}

	return s.refresh(ctx, link)
}

func (s *Store) refresh(ctx context.Context, link db.TelegramLink) (*Session, error) {
	presented, err := s.box.Open(link.RefreshToken)
	if err != nil {
		return nil, err
	}

	res, err := s.auth.Refresh(ctx, connect.NewRequest(&authv1.RefreshRequest{
		RefreshToken: presented,
	}))
	if err != nil {
		if connect.CodeOf(err) == connect.CodeUnauthenticated {
			if _, delErr := s.q.DeleteLink(ctx, db.DeleteLinkParams{
				Bot: link.Bot, TelegramUserID: link.TelegramUserID,
			}); delErr != nil {
				return nil, fmt.Errorf("session: drop dead link: %w", delErr)
			}
			return nil, ErrLinkAgain
		}
		return nil, fmt.Errorf("session: refresh: %w", err)
	}

	access, err := s.box.Seal(res.Msg.GetAccessToken())
	if err != nil {
		return nil, err
	}
	refresh, err := s.box.Seal(res.Msg.GetRefreshToken())
	if err != nil {
		return nil, err
	}

	if _, err := s.q.UpdateLinkTokens(ctx, db.UpdateLinkTokensParams{
		Bot:             link.Bot,
		TelegramUserID:  link.TelegramUserID,
		AccessToken:     access,
		AccessExpiresAt: pgconv.TimestampFrom(s.expiry(res.Msg.GetExpiresIn())),
		RefreshToken:    refresh,
	}); err != nil {
		return nil, fmt.Errorf("session: store refreshed tokens: %w", err)
	}

	return newSession(pgconv.UUIDString(link.UserID), res.Msg.GetAccessToken()), nil
}

func (s *Store) ExpireAccess(ctx context.Context, bot string, telegramUserID int64) error {
	if _, err := s.q.ExpireLinkAccess(ctx, db.ExpireLinkAccessParams{
		Bot: bot, TelegramUserID: telegramUserID,
	}); err != nil {
		return fmt.Errorf("session: expire access token: %w", err)
	}
	return nil
}

func newSession(userID, accessToken string) *Session {
	return &Session{
		UserID:      userID,
		AccessToken: accessToken,
		Locale:      localeFromToken(accessToken),
	}
}

func localeFromToken(accessToken string) string {
	parts := strings.Split(accessToken, ".")
	if len(parts) != 3 {
		return ""
	}
	body, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var claims struct {
		Locale string `json:"locale"`
	}
	if err := json.Unmarshal(body, &claims); err != nil {
		return ""
	}
	return claims.Locale
}

func (s *Store) Unlink(ctx context.Context, bot string, telegramUserID int64) error {
	link, err := s.q.GetLink(ctx, db.GetLinkParams{Bot: bot, TelegramUserID: telegramUserID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotLinked
		}
		return fmt.Errorf("session: get link: %w", err)
	}

	if presented, err := s.box.Open(link.RefreshToken); err == nil {
		if _, err := s.auth.Logout(ctx, connect.NewRequest(&authv1.LogoutRequest{
			RefreshToken: presented,
		})); err != nil {
			return fmt.Errorf("session: logout: %w", err)
		}
	}

	if _, err := s.q.DeleteLink(ctx, db.DeleteLinkParams{
		Bot: bot, TelegramUserID: telegramUserID,
	}); err != nil {
		return fmt.Errorf("session: delete link: %w", err)
	}
	return nil
}

func (s *Store) expiry(expiresIn int64) time.Time {
	if expiresIn <= 0 {
		expiresIn = int64((15 * time.Minute).Seconds())
	}
	return s.now().Add(time.Duration(expiresIn) * time.Second)
}
