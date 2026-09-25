package handler

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	fmauth "github.com/nnc/family-manager/libs/go/auth"
	"github.com/nnc/family-manager/libs/go/database/pgconv"
	familyv1 "github.com/nnc/family-manager/sdk/go/family/v1"
	"github.com/nnc/family-manager/services/family/db"
)

const (
	DefaultLocale   = "uk"
	maxTimezoneSize = 64
)

var supportedLocales = map[string]struct{}{"uk": {}, "en": {}}

func invalid(msg string) error {
	return connect.NewError(connect.CodeInvalidArgument, errors.New(msg))
}

func (h *Handler) GetUserSettings(
	ctx context.Context, req *connect.Request[familyv1.GetUserSettingsRequest],
) (*connect.Response[familyv1.GetUserSettingsResponse], error) {
	userID, err := pgconv.UUID(strings.TrimSpace(req.Msg.GetUserId()))
	if err != nil || !userID.Valid {
		return nil, invalid("user_id is required")
	}

	settings, err := h.settingsOf(ctx, userID)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&familyv1.GetUserSettingsResponse{Settings: settings}), nil
}

func (h *Handler) GetMySettings(
	ctx context.Context, _ *connect.Request[familyv1.GetMySettingsRequest],
) (*connect.Response[familyv1.GetMySettingsResponse], error) {
	claims, err := fmauth.Require(ctx)
	if err != nil {
		return nil, err
	}

	userID, err := pgconv.UUID(claims.UserID)
	if err != nil || !userID.Valid {
		return nil, connect.NewError(connect.CodeUnauthenticated,
			errors.New("token has no usable subject"))
	}

	settings, err := h.settingsOf(ctx, userID)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&familyv1.GetMySettingsResponse{Settings: settings}), nil
}

func (h *Handler) UpdateMySettings(
	ctx context.Context, req *connect.Request[familyv1.UpdateMySettingsRequest],
) (*connect.Response[familyv1.UpdateMySettingsResponse], error) {
	claims, err := fmauth.Require(ctx)
	if err != nil {
		return nil, err
	}

	userID, err := pgconv.UUID(claims.UserID)
	if err != nil || !userID.Valid {
		return nil, connect.NewError(connect.CodeUnauthenticated,
			errors.New("token has no usable subject"))
	}

	locale := strings.ToLower(strings.TrimSpace(req.Msg.GetLocale()))
	if locale != "" {
		if _, ok := supportedLocales[locale]; !ok {
			return nil, invalid("unsupported locale")
		}
	}

	timezone := strings.TrimSpace(req.Msg.GetTimezone())
	if len(timezone) > maxTimezoneSize {
		return nil, invalid("timezone is too long")
	}

	row, err := h.q.UpsertUserSettings(ctx, db.UpsertUserSettingsParams{
		UserID:   userID,
		Locale:   locale,
		Timezone: timezone,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "upsert user settings")
	}

	return connect.NewResponse(&familyv1.UpdateMySettingsResponse{
		Settings: toProtoSettings(row),
	}), nil
}

func (h *Handler) settingsOf(ctx context.Context, userID pgtype.UUID) (*familyv1.UserSettings, error) {
	row, err := h.q.GetUserSettings(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &familyv1.UserSettings{
				UserId: pgconv.UUIDString(userID),
				Locale: DefaultLocale,
			}, nil
		}
		return nil, h.internal(ctx, err, "get user settings")
	}
	return toProtoSettings(row), nil
}

func toProtoSettings(row db.UserSetting) *familyv1.UserSettings {
	locale := row.Locale
	if locale == "" {
		locale = DefaultLocale
	}
	return &familyv1.UserSettings{
		UserId:    pgconv.UUIDString(row.UserID),
		Locale:    locale,
		Timezone:  row.Timezone,
		UpdatedAt: pgconv.Timestamp(row.UpdatedAt),
	}
}
