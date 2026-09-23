package handler

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"

	tasksv1 "github.com/nnc/family-manager/sdk/go/tasks/v1"
	"github.com/nnc/family-manager/services/tasks/db"
)

func (h *Handler) GetFamilySettings(
	ctx context.Context, req *connect.Request[tasksv1.GetFamilySettingsRequest],
) (*connect.Response[tasksv1.GetFamilySettingsResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.touchKnownMember(ctx, c); err != nil {
		return nil, h.internal(ctx, err, "touch known member")
	}

	deviceTZ := req.Msg.GetDeviceTimezone()
	row, err := h.q.GetFamilySettings(ctx, c.familyID)
	if errors.Is(err, pgx.ErrNoRows) {
		loc := time.UTC
		if deviceTZ != "" {
			if l, lerr := time.LoadLocation(deviceTZ); lerr == nil {
				loc = l
			}
		}
		row = db.FamilySetting{
			FamilyID:   c.familyID,
			Timezone:   loc.String(),
			DigestTime: "08:00",
			CreatedAt:  pgTimestamptz(h.now()),
			UpdatedAt:  pgTimestamptz(h.now()),
		}
	} else if err != nil {
		return nil, h.internal(ctx, err, "get family settings")
	}

	return connect.NewResponse(&tasksv1.GetFamilySettingsResponse{
		Settings: &tasksv1.FamilySettings{
			Timezone:   row.Timezone,
			DigestTime: row.DigestTime,
		},
	}), nil
}

func (h *Handler) SetFamilySettings(
	ctx context.Context, req *connect.Request[tasksv1.SetFamilySettingsRequest],
) (*connect.Response[tasksv1.SetFamilySettingsResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.touchKnownMember(ctx, c); err != nil {
		return nil, h.internal(ctx, err, "touch known member")
	}

	tz := req.Msg.GetTimezone()
	if tz == "" {
		return nil, invalid("timezone is required")
	}
	if _, err := time.LoadLocation(tz); err != nil {
		return nil, invalid("invalid timezone: %v", err)
	}

	row, err := h.q.SetFamilyTimezone(ctx, db.SetFamilyTimezoneParams{
		FamilyID: c.familyID,
		Timezone: tz,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "set family timezone")
	}

	return connect.NewResponse(&tasksv1.SetFamilySettingsResponse{
		Settings: &tasksv1.FamilySettings{
			Timezone:   row.Timezone,
			DigestTime: row.DigestTime,
		},
	}), nil
}