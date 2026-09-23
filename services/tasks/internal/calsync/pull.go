package calsync

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/proto"

	"github.com/nnc/family-manager/libs/go/database/pgconv"
	"github.com/nnc/family-manager/libs/go/events"
	"github.com/nnc/family-manager/services/tasks/db"
	"github.com/nnc/family-manager/services/tasks/internal/crypto"
	"github.com/nnc/family-manager/services/tasks/internal/gcal"
)

type Bus interface {
	Publish(ctx context.Context, subject events.Subject, msg proto.Message) error
}

type PullerOptions struct {
	Queries db.Querier
	Bus     Bus
	Google  gcal.Client
	Box     *crypto.Box
	Tick    time.Duration
	Log     *slog.Logger
}

type Puller struct {
	q      db.Querier
	bus    Bus
	google gcal.Client
	box    *crypto.Box
	tick   time.Duration
	log    *slog.Logger

	mu     sync.Mutex
	tokens map[string]gcal.Token
}

func NewPuller(opts PullerOptions) *Puller {
	return &Puller{
		q:      opts.Queries,
		bus:    opts.Bus,
		google: opts.Google,
		box:    opts.Box,
		tick:   opts.Tick,
		log:    opts.Log,
		tokens: map[string]gcal.Token{},
	}
}

func (p *Puller) Run(ctx context.Context) {
	if p.tick <= 0 {
		p.log.Info("calendar pull sync disabled")
		return
	}
	ticker := time.NewTicker(p.tick)
	defer ticker.Stop()
	p.syncAll(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.syncAll(ctx)
		}
	}
}

func (p *Puller) syncAll(ctx context.Context) {
	conns, err := p.q.ListFamilyGoogleConnections(ctx, pgtype.UUID{})
	if err != nil {
		p.log.Error("list google connections", slog.String("error", err.Error()))
		return
	}

	for _, conn := range conns {
		func() {
			defer func() {
				if r := recover(); r != nil {
					p.log.Error("panic in calendar sync",
						slog.String("family_id", pgconv.UUIDString(conn.FamilyID)),
						slog.String("user_id", pgconv.UUIDString(conn.UserID)),
						slog.Any("panic", r),
					)
				}
			}()
			p.syncConnection(ctx, conn)
		}()
	}
}

func (p *Puller) syncConnection(ctx context.Context, conn db.GoogleConnection) error {
	access, err := p.accessToken(ctx, conn)
	if err != nil {
		p.log.Error("access token", slog.String("error", err.Error()))
		return p.fail(ctx, conn, err)
	}

	changes, err := p.google.ListChanges(ctx, access, conn.CalendarID, conn.SyncToken)
	if errors.Is(err, gcal.ErrSyncTokenExpired) {
		changes, err = p.google.ListChanges(ctx, access, conn.CalendarID, "")
	}
	if err != nil {
		return p.fail(ctx, conn, err)
	}

	for _, e := range changes.Events {
		p.processEvent(ctx, conn, e)
	}

	if changes.NextSyncToken != "" && changes.NextSyncToken != conn.SyncToken {
		if err := p.q.SetGoogleSyncToken(ctx, db.SetGoogleSyncTokenParams{
			FamilyID: conn.FamilyID, UserID: conn.UserID, SyncToken: changes.NextSyncToken,
		}); err != nil {
			p.log.Error("update sync token", slog.String("error", err.Error()))
		}
	}

	return nil
}

func (p *Puller) processEvent(ctx context.Context, conn db.GoogleConnection, e gcal.Event) {
	kind := e.Private[gcal.PropKind]
	itemIDStr := e.Private[gcal.PropItemID]
	familyIDStr := e.Private[gcal.PropFamily]

	if kind == "" || itemIDStr == "" || familyIDStr == "" {
		return
	}

	familyID, err := pgconv.UUID(familyIDStr)
	if err != nil {
		p.log.Warn("invalid family_id in event", slog.String("family_id", familyIDStr))
		return
	}
	itemID, err := pgconv.UUID(itemIDStr)
	if err != nil {
		p.log.Warn("invalid item_id in event", slog.String("item_id", itemIDStr))
		return
	}

	switch kind {
	case "task":
		p.processTaskEvent(ctx, conn, familyID, itemID, e)
	case "birthday":
		p.processBirthdayEvent(ctx, conn, familyID, itemID, e)
	}
}

func (p *Puller) processTaskEvent(ctx context.Context, conn db.GoogleConnection, familyID, itemID pgtype.UUID, e gcal.Event) {
	if e.Cancelled() {
		p.handleTaskDeleted(ctx, conn, familyID, itemID, e.ID)
		return
	}

	task, err := p.q.GetTask(ctx, db.GetTaskParams{ID: itemID, FamilyID: familyID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return
		}
		p.log.Error("get task", slog.String("error", err.Error()))
		return
	}

	if task.Status == "done" {
		return
	}

	dueChanged := false
	if e.AllDay {
		dueOn := pgtype.Date{Time: parseDate(e.StartDate), Valid: true}
		if !task.DueOn.Valid || task.DueOn.Time != dueOn.Time {
			dueChanged = true
		}
	} else {
		dueAt := e.Start
		if !task.DueAt.Valid || task.DueAt.Time != dueAt {
			dueChanged = true
		}
	}

	if dueChanged {
		p.log.Info("task due date changed from Google",
			slog.String("family_id", pgconv.UUIDString(familyID)),
			slog.String("task_id", pgconv.UUIDString(itemID)),
		)
	}
}

func (p *Puller) handleTaskDeleted(ctx context.Context, conn db.GoogleConnection, familyID, itemID pgtype.UUID, eventID string) {
	p.log.Info("task deleted in Google",
		slog.String("family_id", pgconv.UUIDString(familyID)),
		slog.String("task_id", pgconv.UUIDString(itemID)),
	)

	link, err := p.q.GetCalendarLink(ctx, db.GetCalendarLinkParams{
		FamilyID: familyID, UserID: conn.UserID, Kind: "task", ItemID: itemID,
	})
	if err != nil {
		return
	}
	if link.EventID == eventID {
		if err := p.q.DeleteCalendarLink(ctx, db.DeleteCalendarLinkParams{
			FamilyID: familyID, UserID: conn.UserID, Kind: "task", ItemID: itemID,
		}); err != nil {
			p.log.Error("delete calendar link", slog.String("error", err.Error()))
		}
	}
}

func (p *Puller) processBirthdayEvent(ctx context.Context, conn db.GoogleConnection, familyID, itemID pgtype.UUID, e gcal.Event) {
	if e.Cancelled() {
		p.handleBirthdayDeleted(ctx, conn, familyID, itemID, e.ID)
		return
	}

	_, err := p.q.GetBirthday(ctx, db.GetBirthdayParams{ID: itemID, FamilyID: familyID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return
		}
		p.log.Error("get birthday", slog.String("error", err.Error()))
		return
	}
}

func (p *Puller) handleBirthdayDeleted(ctx context.Context, conn db.GoogleConnection, familyID, itemID pgtype.UUID, eventID string) {
	p.log.Info("birthday deleted in Google",
		slog.String("family_id", pgconv.UUIDString(familyID)),
		slog.String("birthday_id", pgconv.UUIDString(itemID)),
	)

	link, err := p.q.GetCalendarLink(ctx, db.GetCalendarLinkParams{
		FamilyID: familyID, UserID: conn.UserID, Kind: "birthday", ItemID: itemID,
	})
	if err != nil {
		return
	}
	if link.EventID == eventID {
		if err := p.q.DeleteCalendarLink(ctx, db.DeleteCalendarLinkParams{
			FamilyID: familyID, UserID: conn.UserID, Kind: "birthday", ItemID: itemID,
		}); err != nil {
			p.log.Error("delete calendar link", slog.String("error", err.Error()))
		}
	}
}

func (p *Puller) accessToken(ctx context.Context, conn db.GoogleConnection) (string, error) {
	key := pgconv.UUIDString(conn.FamilyID) + "|" + pgconv.UUIDString(conn.UserID)
	p.mu.Lock()
	tok, ok := p.tokens[key]
	p.mu.Unlock()
	if ok && p.now().Add(time.Minute).Before(tok.Expiry) {
		return tok.AccessToken, nil
	}

	refresh, err := p.box.Open(conn.RefreshToken, crypto.Owner(pgconv.UUIDString(conn.FamilyID), pgconv.UUIDString(conn.UserID)))
	if err != nil {
		return "", fmt.Errorf("open refresh token: %w", err)
	}
	tok, err = p.google.Refresh(ctx, refresh)
	if err != nil {
		return "", err
	}
	p.mu.Lock()
	p.tokens[key] = tok
	p.mu.Unlock()
	return tok.AccessToken, nil
}

func (p *Puller) fail(ctx context.Context, conn db.GoogleConnection, err error) error {
	text := "Could not sync Google Calendar. We will retry."
	switch {
	case errors.Is(err, gcal.ErrForbidden):
		text = "Google Calendar no longer lets us write to this calendar. Pick another calendar."
	case errors.Is(err, gcal.ErrUnauthorized), errors.Is(err, gcal.ErrInsufficientScope):
		text = "Google access was revoked. Connect Google Calendar again."
		p.mu.Lock()
		delete(p.tokens, pgconv.UUIDString(conn.FamilyID)+"|"+pgconv.UUIDString(conn.UserID))
		p.mu.Unlock()
	}
	if serr := p.q.SetGoogleError(ctx, db.SetGoogleErrorParams{
		FamilyID: conn.FamilyID, UserID: conn.UserID, LastError: text,
	}); serr != nil {
		p.log.WarnContext(ctx, "record google error", slog.String("error", serr.Error()))
	}
	return fmt.Errorf("calsync: %s/%s: %w",
		pgconv.UUIDString(conn.FamilyID), pgconv.UUIDString(conn.UserID), err)
}

func (p *Puller) now() time.Time {
	return time.Now()
}

func parseDate(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}