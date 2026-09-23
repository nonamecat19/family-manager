package handler

import (
	"context"
	"errors"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nnc/family-manager/libs/go/database/pgconv"
	tasksv1 "github.com/nnc/family-manager/sdk/go/tasks/v1"
	"github.com/nnc/family-manager/services/tasks/db"
	"github.com/nnc/family-manager/services/tasks/internal/crypto"
)

func (h *Handler) GetGoogleConnection(
	ctx context.Context, req *connect.Request[tasksv1.GetGoogleConnectionRequest],
) (*connect.Response[tasksv1.GetGoogleConnectionResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.touchKnownMember(ctx, c); err != nil {
		return nil, h.internal(ctx, err, "touch known member")
	}

	conn, err := h.q.GetGoogleConnection(ctx, db.GetGoogleConnectionParams{
		FamilyID: c.familyID, UserID: c.userID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return connect.NewResponse(&tasksv1.GetGoogleConnectionResponse{
			Connection: &tasksv1.GoogleConnection{Configured: false, Connected: false},
		}), nil
	}
	if err != nil {
		return nil, h.internal(ctx, err, "get google connection")
	}

	return connect.NewResponse(&tasksv1.GetGoogleConnectionResponse{
		Connection: toProtoGoogleConnection(conn),
	}), nil
}

func (h *Handler) ConnectGoogle(
	ctx context.Context, req *connect.Request[tasksv1.ConnectGoogleRequest],
) (*connect.Response[tasksv1.ConnectGoogleResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.touchKnownMember(ctx, c); err != nil {
		return nil, h.internal(ctx, err, "touch known member")
	}

	code := req.Msg.GetCode()
	verifier := req.Msg.GetCodeVerifier()
	redirectURI := req.Msg.GetRedirectUri()
	if code == "" || verifier == "" || redirectURI == "" {
		return nil, invalid("code, code_verifier and redirect_uri are required")
	}

	if h.calendar == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			errors.New("Google Calendar not configured on server"))
	}

	gcalClient := h.calendar.GoogleClient()
	if gcalClient == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			errors.New("Google Calendar not configured on server"))
	}

	token, err := gcalClient.Exchange(ctx, code, verifier, redirectURI)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}

	box := h.calendar.Box()
	if box == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			errors.New("crypto box not configured"))
	}
	sealed, err := box.Seal(token.RefreshToken, crypto.Owner(pgconv.UUIDString(c.familyID), pgconv.UUIDString(c.userID)))
	if err != nil {
		return nil, h.internal(ctx, err, "seal refresh token")
	}

	conn, err := h.q.UpsertGoogleConnection(ctx, db.UpsertGoogleConnectionParams{
		FamilyID:     c.familyID,
		UserID:       c.userID,
		Email:        token.Email,
		RefreshToken: sealed,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "save google connection")
	}

	return connect.NewResponse(&tasksv1.ConnectGoogleResponse{
		Connection: toProtoGoogleConnection(conn),
	}), nil
}

func (h *Handler) DisconnectGoogle(
	ctx context.Context, req *connect.Request[tasksv1.DisconnectGoogleRequest],
) (*connect.Response[tasksv1.DisconnectGoogleResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.touchKnownMember(ctx, c); err != nil {
		return nil, h.internal(ctx, err, "touch known member")
	}

	conn, err := h.q.GetGoogleConnection(ctx, db.GetGoogleConnectionParams{
		FamilyID: c.familyID, UserID: c.userID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return connect.NewResponse(&tasksv1.DisconnectGoogleResponse{}), nil
	}
	if err != nil {
		return nil, h.internal(ctx, err, "get google connection")
	}

	if h.calendar != nil && conn.CalendarID != "" {
		gcalClient := h.calendar.GoogleClient()
		box := h.calendar.Box()
		if gcalClient != nil && box != nil {
			refresh, err := box.Open(conn.RefreshToken, crypto.Owner(pgconv.UUIDString(c.familyID), pgconv.UUIDString(c.userID)))
			if err == nil {
				access, err := gcalClient.Refresh(ctx, refresh)
				if err == nil {
					_ = gcalClient.Revoke(ctx, access.AccessToken)
				}
			}
		}
	}

	_, err = h.q.DeleteGoogleConnection(ctx, db.DeleteGoogleConnectionParams{
		FamilyID: c.familyID, UserID: c.userID,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "delete google connection")
	}

	if err := h.q.DeleteCalendarLinksForUser(ctx, db.DeleteCalendarLinksForUserParams{
		FamilyID: c.familyID, UserID: c.userID,
	}); err != nil {
		return nil, h.internal(ctx, err, "delete calendar links")
	}

	return connect.NewResponse(&tasksv1.DisconnectGoogleResponse{}), nil
}

func (h *Handler) ListGoogleCalendars(
	ctx context.Context, req *connect.Request[tasksv1.ListGoogleCalendarsRequest],
) (*connect.Response[tasksv1.ListGoogleCalendarsResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.touchKnownMember(ctx, c); err != nil {
		return nil, h.internal(ctx, err, "touch known member")
	}

	conn, err := h.q.GetGoogleConnection(ctx, db.GetGoogleConnectionParams{
		FamilyID: c.familyID, UserID: c.userID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			errors.New("Google not connected"))
	}
	if err != nil {
		return nil, h.internal(ctx, err, "get google connection")
	}

	if h.calendar == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			errors.New("Google Calendar not configured on server"))
	}

	gcalClient := h.calendar.GoogleClient()
	box := h.calendar.Box()
	if gcalClient == nil || box == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			errors.New("Google Calendar not configured on server"))
	}

	refresh, err := box.Open(conn.RefreshToken, crypto.Owner(pgconv.UUIDString(c.familyID), pgconv.UUIDString(c.userID)))
	if err != nil {
		return nil, h.internal(ctx, err, "open refresh token")
	}

	token, err := gcalClient.Refresh(ctx, refresh)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}

	calendars, err := gcalClient.ListCalendars(ctx, token.AccessToken)
	if err != nil {
		return nil, h.internal(ctx, err, "list google calendars")
	}

	pbCalendars := make([]*tasksv1.GoogleCalendar, 0, len(calendars))
	for _, cal := range calendars {
		pbCalendars = append(pbCalendars, &tasksv1.GoogleCalendar{
			Id:      cal.ID,
			Name:    cal.Name,
			Primary: cal.Primary,
		})
	}

	return connect.NewResponse(&tasksv1.ListGoogleCalendarsResponse{Calendars: pbCalendars}), nil
}

func (h *Handler) SetGoogleCalendar(
	ctx context.Context, req *connect.Request[tasksv1.SetGoogleCalendarRequest],
) (*connect.Response[tasksv1.SetGoogleCalendarResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.touchKnownMember(ctx, c); err != nil {
		return nil, h.internal(ctx, err, "touch known member")
	}

	calendarID := req.Msg.GetCalendarId()
	if calendarID == "" {
		return nil, invalid("calendar_id is required")
	}

	_, err = h.q.GetGoogleConnection(ctx, db.GetGoogleConnectionParams{
		FamilyID: c.familyID, UserID: c.userID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			errors.New("Google not connected"))
	}
	if err != nil {
		return nil, h.internal(ctx, err, "get google connection")
	}

	updated, err := h.q.SetGoogleCalendar(ctx, db.SetGoogleCalendarParams{
		FamilyID: c.familyID, UserID: c.userID, CalendarID: calendarID, CalendarName: "",
	})
	if err != nil {
		return nil, h.internal(ctx, err, "set google calendar")
	}

	if h.calendar != nil {
		if err := h.calendar.SwitchCalendar(ctx, c.familyID, c.userID); err != nil {
			h.log.WarnContext(ctx, "switch calendar", slog.String("error", err.Error()))
		}
	}

	return connect.NewResponse(&tasksv1.SetGoogleCalendarResponse{
		Connection: toProtoGoogleConnection(updated),
	}), nil
}

func toProtoGoogleConnection(conn db.GoogleConnection) *tasksv1.GoogleConnection {
	return &tasksv1.GoogleConnection{
		Configured:    true,
		Connected:     conn.CalendarID != "",
		ClientId:      "",
		Email:         conn.Email,
		CalendarId:    conn.CalendarID,
		CalendarName:  conn.CalendarName,
		LastSyncedAt:  timestamppb.New(conn.LastSyncedAt.Time),
		LastError:     conn.LastError,
	}
}