package session

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
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
	ErrTaken     = errors.New("session: this telegram account is linked to another user")
)

type Session struct {
	UserID      string
	AccessToken string
	Locale      string
}

type Store struct {
	q     db.Querier
	box   *secret.Box
	auth  authv1connect.AuthServiceClient
	now   func() time.Time
	locks sync.Map
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
	ctx context.Context, linkToken string, from telegram.User, chatID int64,
) (*Session, error) {
	res, err := s.auth.RedeemLinkToken(ctx, connect.NewRequest(&authv1.RedeemLinkTokenRequest{
		Token:      linkToken,
		Provider:   provider,
		ExternalId: fmt.Sprint(from.ID),
	}))
	if err != nil {
		switch connect.CodeOf(err) {
		case connect.CodeUnauthenticated:
			return nil, ErrLinkAgain
		case connect.CodeAlreadyExists:
			return nil, ErrTaken
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

func (s *Store) Session(ctx context.Context, telegramUserID int64) (*Session, error) {
	if current, _, err := s.cached(ctx, telegramUserID); err != nil || current != nil {
		return current, err
	}

	mu := s.lockFor(telegramUserID)
	mu.Lock()
	defer mu.Unlock()

	current, link, err := s.cached(ctx, telegramUserID)
	if err != nil || current != nil {
		return current, err
	}
	return s.refresh(ctx, *link)
}

func (s *Store) cached(ctx context.Context, telegramUserID int64) (*Session, *db.TelegramLink, error) {
	link, err := s.q.GetLink(ctx, telegramUserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, ErrNotLinked
		}
		return nil, nil, fmt.Errorf("session: get link: %w", err)
	}

	if link.AccessExpiresAt.Valid && link.AccessExpiresAt.Time.After(s.now().Add(refreshSkew)) {
		access, err := s.box.Open(link.AccessToken)
		if err != nil {
			return nil, nil, err
		}
		return newSession(pgconv.UUIDString(link.UserID), access), &link, nil
	}
	return nil, &link, nil
}

func (s *Store) lockFor(telegramUserID int64) *sync.Mutex {
	mu, _ := s.locks.LoadOrStore(telegramUserID, &sync.Mutex{})
	return mu.(*sync.Mutex)
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
			if _, delErr := s.q.DeleteLinkWithToken(ctx, db.DeleteLinkWithTokenParams{
				TelegramUserID: link.TelegramUserID,
				RefreshToken:   link.RefreshToken,
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

	stored, err := s.q.UpdateLinkTokens(ctx, db.UpdateLinkTokensParams{
		TelegramUserID:       link.TelegramUserID,
		PreviousRefreshToken: link.RefreshToken,
		AccessToken:          access,
		AccessExpiresAt:      pgconv.TimestampFrom(s.expiry(res.Msg.GetExpiresIn())),
		RefreshToken:         refresh,
	})
	if err != nil {
		return nil, fmt.Errorf("session: store refreshed tokens: %w", err)
	}
	if stored == 0 {
		return nil, ErrLinkAgain
	}

	return newSession(pgconv.UUIDString(link.UserID), res.Msg.GetAccessToken()), nil
}

func (s *Store) ExpireAccess(ctx context.Context, telegramUserID int64) error {
	if _, err := s.q.ExpireLinkAccess(ctx, telegramUserID); err != nil {
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

func (s *Store) Unlink(ctx context.Context, telegramUserID int64) error {
	current, err := s.Session(ctx, telegramUserID)
	switch {
	case errors.Is(err, ErrNotLinked):
		return ErrNotLinked
	case errors.Is(err, ErrLinkAgain):
		return nil
	case err != nil:
		return err
	}

	req := connect.NewRequest(&authv1.UnlinkRequest{
		Provider:   provider,
		ExternalId: fmt.Sprint(telegramUserID),
	})
	req.Header().Set("Authorization", "Bearer "+current.AccessToken)
	if _, err := s.auth.Unlink(ctx, req); err != nil && connect.CodeOf(err) != connect.CodeNotFound {
		return fmt.Errorf("session: unlink: %w", err)
	}

	userID, err := pgconv.UUID(current.UserID)
	if err != nil {
		return fmt.Errorf("session: malformed user id: %w", err)
	}
	if _, err := s.q.DeleteLinkForUser(ctx, db.DeleteLinkForUserParams{
		TelegramUserID: telegramUserID, UserID: userID,
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
