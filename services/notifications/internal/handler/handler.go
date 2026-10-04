package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"unicode/utf8"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgtype"

	fmauth "github.com/nnc/family-manager/libs/go/auth"
	"github.com/nnc/family-manager/libs/go/database/pgconv"
	"github.com/nnc/family-manager/libs/go/rpc"
	notificationsv1 "github.com/nnc/family-manager/sdk/go/notifications/v1"
	"github.com/nnc/family-manager/services/notifications/db"
	"github.com/nnc/family-manager/services/notifications/internal/topics"
)

const (
	maxTokenBytes  = 256
	maxDeviceRunes = 128
	maxMuted       = 32
)

type Tx interface {
	InTx(ctx context.Context, fn func(q db.Querier) error) error
}

type Options struct {
	Queries db.Querier
	Tx      Tx
	Log     *slog.Logger
}

type Handler struct {
	q   db.Querier
	tx  Tx
	log *slog.Logger
}

func New(opts Options) *Handler {
	h := &Handler{q: opts.Queries, tx: opts.Tx, log: opts.Log}
	if h.log == nil {
		h.log = slog.Default()
	}
	return h
}

func (h *Handler) RegisterPushToken(
	ctx context.Context, req *connect.Request[notificationsv1.RegisterPushTokenRequest],
) (*connect.Response[notificationsv1.RegisterPushTokenResponse], error) {
	claims, userID, err := caller(ctx)
	if err != nil {
		return nil, err
	}

	token := strings.TrimSpace(req.Msg.GetToken())
	if !ValidToken(token) {
		return nil, invalid("token must be an Expo push token")
	}
	platform, ok := platformFromProto(req.Msg.GetPlatform())
	if !ok {
		return nil, invalid("platform is required")
	}
	app, ok := AppFromProto(req.Msg.GetApp())
	if !ok {
		return nil, invalid("app is required")
	}
	deviceID := strings.TrimSpace(req.Msg.GetDeviceId())
	if utf8.RuneCountInString(deviceID) > maxDeviceRunes {
		return nil, invalid(fmt.Sprintf("device_id must be at most %d characters", maxDeviceRunes))
	}

	var familyID pgtype.UUID
	if claims.FamilyID != "" {
		if id, err := pgconv.UUID(claims.FamilyID); err == nil {
			familyID = id
		}
	}

	rows, err := h.q.UpsertPushToken(ctx, db.UpsertPushTokenParams{
		Token:    token,
		UserID:   userID,
		FamilyID: familyID,
		Platform: platform,
		App:      app,
		DeviceID: deviceID,
	})
	if err != nil {
		return nil, rpc.Internal(ctx, h.log, err, "upsert push token")
	}
	if rows == 0 {
		return nil, connect.NewError(connect.CodePermissionDenied,
			errors.New("this push token is registered to another account on a different device"))
	}
	return connect.NewResponse(&notificationsv1.RegisterPushTokenResponse{}), nil
}

func (h *Handler) UnregisterPushToken(
	ctx context.Context, req *connect.Request[notificationsv1.UnregisterPushTokenRequest],
) (*connect.Response[notificationsv1.UnregisterPushTokenResponse], error) {
	_, userID, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	token := strings.TrimSpace(req.Msg.GetToken())
	if token == "" || len(token) > maxTokenBytes {
		return nil, invalid("token is required")
	}
	if _, err := h.q.DeleteUserPushToken(ctx, db.DeleteUserPushTokenParams{
		Token: token, UserID: userID,
	}); err != nil {
		return nil, rpc.Internal(ctx, h.log, err, "delete push token")
	}
	return connect.NewResponse(&notificationsv1.UnregisterPushTokenResponse{}), nil
}

func (h *Handler) GetPreferences(
	ctx context.Context, _ *connect.Request[notificationsv1.GetPreferencesRequest],
) (*connect.Response[notificationsv1.GetPreferencesResponse], error) {
	_, userID, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	muted, err := h.q.ListMutes(ctx, userID)
	if err != nil {
		return nil, rpc.Internal(ctx, h.log, err, "list mutes")
	}
	return connect.NewResponse(&notificationsv1.GetPreferencesResponse{
		Muted:  nonNil(muted),
		Topics: catalog(),
	}), nil
}

func (h *Handler) SetPreferences(
	ctx context.Context, req *connect.Request[notificationsv1.SetPreferencesRequest],
) (*connect.Response[notificationsv1.SetPreferencesResponse], error) {
	_, userID, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	if len(req.Msg.GetMuted()) > maxMuted {
		return nil, invalid(fmt.Sprintf("at most %d muted entries", maxMuted))
	}

	muted := make([]string, 0, len(req.Msg.GetMuted()))
	for _, m := range req.Msg.GetMuted() {
		m = strings.TrimSpace(m)
		if !topics.Valid(m) {
			return nil, invalid(fmt.Sprintf("unknown topic %q", m))
		}
		if !slices.Contains(muted, m) {
			muted = append(muted, m)
		}
	}
	slices.Sort(muted)

	err = h.tx.InTx(ctx, func(q db.Querier) error {
		if err := q.DeleteMutes(ctx, userID); err != nil {
			return fmt.Errorf("delete mutes: %w", err)
		}
		for _, m := range muted {
			if err := q.InsertMute(ctx, db.InsertMuteParams{UserID: userID, Topic: m}); err != nil {
				return fmt.Errorf("insert mute: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, rpc.Internal(ctx, h.log, err, "set preferences")
	}
	return connect.NewResponse(&notificationsv1.SetPreferencesResponse{Muted: muted}), nil
}

func ValidToken(token string) bool {
	if len(token) == 0 || len(token) > maxTokenBytes || !strings.HasSuffix(token, "]") {
		return false
	}
	return strings.HasPrefix(token, "ExponentPushToken[") || strings.HasPrefix(token, "ExpoPushToken[")
}

func AppFromProto(app notificationsv1.App) (string, bool) {
	switch app {
	case notificationsv1.App_APP_NOTES:
		return "notes", true
	case notificationsv1.App_APP_FINANCE:
		return "finance", true
	case notificationsv1.App_APP_RECIPES:
		return "recipes", true
	default:
		return "", false
	}
}

func platformFromProto(p notificationsv1.Platform) (string, bool) {
	switch p {
	case notificationsv1.Platform_PLATFORM_IOS:
		return "ios", true
	case notificationsv1.Platform_PLATFORM_ANDROID:
		return "android", true
	default:
		return "", false
	}
}

func caller(ctx context.Context) (*fmauth.Claims, pgtype.UUID, error) {
	claims, err := fmauth.Require(ctx)
	if err != nil {
		return nil, pgtype.UUID{}, err
	}
	userID, err := pgconv.UUID(claims.UserID)
	if err != nil {
		return nil, pgtype.UUID{}, connect.NewError(connect.CodeUnauthenticated,
			errors.New("token subject is not a user id"))
	}
	return claims, userID, nil
}

func catalog() []*notificationsv1.Topic {
	out := make([]*notificationsv1.Topic, 0, len(topics.All))
	for _, t := range topics.All {
		out = append(out, &notificationsv1.Topic{Key: t.Key, Domain: t.Domain})
	}
	return out
}

func invalid(msg string) error {
	return connect.NewError(connect.CodeInvalidArgument, errors.New(msg))
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
