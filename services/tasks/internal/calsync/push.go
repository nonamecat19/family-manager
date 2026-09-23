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

	"github.com/nnc/family-manager/libs/go/database/pgconv"
	"github.com/nnc/family-manager/services/tasks/db"
	"github.com/nnc/family-manager/services/tasks/internal/crypto"
	"github.com/nnc/family-manager/services/tasks/internal/gcal"
)

const (
	ErrTextReadOnly  = "Google Calendar no longer lets us write to this calendar. Pick another calendar."
	ErrTextReconnect = "Google access was revoked. Connect Google Calendar again."
	ErrTextSync      = "Could not update Google Calendar. We will retry."
)

type Store interface {
	GetTask(ctx context.Context, arg db.GetTaskParams) (db.Task, error)
	GetBirthday(ctx context.Context, arg db.GetBirthdayParams) (db.Birthday, error)
	GetFamilySettings(ctx context.Context, familyID pgtype.UUID) (db.FamilySetting, error)
	GetGoogleConnection(ctx context.Context, arg db.GetGoogleConnectionParams) (db.GoogleConnection, error)
	ListFamilyGoogleConnections(ctx context.Context, familyID pgtype.UUID) ([]db.GoogleConnection, error)
	SetGoogleError(ctx context.Context, arg db.SetGoogleErrorParams) error
	GetCalendarLink(ctx context.Context, arg db.GetCalendarLinkParams) (db.CalendarLink, error)
	ListCalendarLinksForItem(ctx context.Context, arg db.ListCalendarLinksForItemParams) ([]db.CalendarLink, error)
	ListCalendarLinksForUser(ctx context.Context, arg db.ListCalendarLinksForUserParams) ([]db.CalendarLink, error)
	ListOrphanCalendarLinks(ctx context.Context, arg db.ListOrphanCalendarLinksParams) ([]db.CalendarLink, error)
	CalendarLinkedInFamily(ctx context.Context, arg db.CalendarLinkedInFamilyParams) (bool, error)
	UpsertCalendarLink(ctx context.Context, arg db.UpsertCalendarLinkParams) error
	DeleteCalendarLink(ctx context.Context, arg db.DeleteCalendarLinkParams) error
}

type Options struct {
	Store  Store
	Google gcal.Client
	Box    *crypto.Box
	Log    *slog.Logger
	Now    func() time.Time
}

type Pusher struct {
	q      Store
	google gcal.Client
	box    *crypto.Box
	log    *slog.Logger
	now    func() time.Time

	mu     sync.Mutex
	tokens map[string]gcal.Token
	items  map[string]*itemLock
}

type itemLock struct {
	mu    sync.Mutex
	users int
}

func NewPusher(opts Options) *Pusher {
	p := &Pusher{
		q: opts.Store, google: opts.Google, box: opts.Box, log: opts.Log, now: opts.Now,
		tokens: map[string]gcal.Token{},
		items:  map[string]*itemLock{},
	}
	if p.log == nil {
		p.log = slog.Default()
	}
	if p.now == nil {
		p.now = time.Now
	}
	return p
}

type item struct {
	kind     string
	familyID pgtype.UUID
	itemID   pgtype.UUID
	event    gcal.Event
	wanted   bool
}

func (p *Pusher) PushTask(ctx context.Context, familyID, taskID pgtype.UUID) error {
	it := item{kind: KindTask, familyID: familyID, itemID: taskID}
	defer p.lockItem(it)()
	return p.pushTask(ctx, it, false)
}

func (p *Pusher) MutateTask(ctx context.Context, familyID, taskID pgtype.UUID, write func() (bool, error)) error {
	it := item{kind: KindTask, familyID: familyID, itemID: taskID}
	defer p.lockItem(it)()
	restore, err := write()
	if err != nil {
		return err
	}
	if err := p.pushTask(ctx, it, restore); err != nil {
		p.log.WarnContext(ctx, "push task to calendar", slog.String("error", err.Error()))
	}
	return nil
}

func (p *Pusher) pushTask(ctx context.Context, it item, restore bool) error {
	familyID, taskID := it.familyID, it.itemID
	task, err := p.q.GetTask(ctx, db.GetTaskParams{ID: taskID, FamilyID: familyID})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
	case err != nil:
		return fmt.Errorf("calsync: load task: %w", err)
	default:
		loc, err := p.location(ctx, familyID)
		if err != nil {
			return err
		}
		it.event, it.wanted = TaskEvent(task, loc)
	}
	return p.push(ctx, it, restore)
}

func (p *Pusher) PushBirthday(ctx context.Context, familyID, birthdayID pgtype.UUID) error {
	it := item{kind: KindBirthday, familyID: familyID, itemID: birthdayID}
	defer p.lockItem(it)()
	return p.pushBirthday(ctx, it)
}

func (p *Pusher) MutateBirthday(ctx context.Context, familyID, birthdayID pgtype.UUID, write func() error) error {
	it := item{kind: KindBirthday, familyID: familyID, itemID: birthdayID}
	defer p.lockItem(it)()
	if err := write(); err != nil {
		return err
	}
	if err := p.pushBirthday(ctx, it); err != nil {
		p.log.WarnContext(ctx, "push birthday to calendar", slog.String("error", err.Error()))
	}
	return nil
}

func (p *Pusher) pushBirthday(ctx context.Context, it item) error {
	familyID, birthdayID := it.familyID, it.itemID
	b, err := p.q.GetBirthday(ctx, db.GetBirthdayParams{ID: birthdayID, FamilyID: familyID})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
	case err != nil:
		return fmt.Errorf("calsync: load birthday: %w", err)
	default:
		loc, err := p.location(ctx, familyID)
		if err != nil {
			return err
		}
		it.event, it.wanted = BirthdayEvent(b, p.now(), loc), true
	}
	return p.push(ctx, it, false)
}

func (p *Pusher) SweepOrphans(ctx context.Context, familyID, userID pgtype.UUID) error {
	links, err := p.q.ListOrphanCalendarLinks(ctx, db.ListOrphanCalendarLinksParams{FamilyID: familyID, UserID: userID})
	if err != nil {
		return fmt.Errorf("calsync: list orphan links: %w", err)
	}
	var errs []error
	for _, l := range links {
		func() {
			defer p.lockItem(item{kind: l.Kind, familyID: l.FamilyID, itemID: l.ItemID})()
			current, err := p.q.GetCalendarLink(ctx, db.GetCalendarLinkParams{
				FamilyID: l.FamilyID, UserID: l.UserID, Kind: l.Kind, ItemID: l.ItemID,
			})
			if errors.Is(err, pgx.ErrNoRows) {
				return
			}
			if err != nil {
				errs = append(errs, fmt.Errorf("calsync: get orphan link: %w", err))
				return
			}
			if current.EventID != l.EventID || current.CalendarID != l.CalendarID {
				return
			}
			switch l.Kind {
			case KindTask:
				task, err := p.q.GetTask(ctx, db.GetTaskParams{ID: l.ItemID, FamilyID: l.FamilyID})
				if err != nil && !errors.Is(err, pgx.ErrNoRows) {
					errs = append(errs, fmt.Errorf("calsync: recheck orphan task: %w", err))
					return
				}
				if err == nil && task.DueOn.Valid {
					return
				}
			case KindBirthday:
				_, err := p.q.GetBirthday(ctx, db.GetBirthdayParams{ID: l.ItemID, FamilyID: l.FamilyID})
				if err == nil {
					return
				}
				if !errors.Is(err, pgx.ErrNoRows) {
					errs = append(errs, fmt.Errorf("calsync: recheck orphan birthday: %w", err))
					return
				}
			}
			errs = append(errs, p.unlink(ctx, l))
		}()
	}
	return errors.Join(errs...)
}

func (p *Pusher) SwitchCalendar(ctx context.Context, familyID, userID pgtype.UUID) error {
	links, err := p.q.ListCalendarLinksForUser(ctx, db.ListCalendarLinksForUserParams{FamilyID: familyID, UserID: userID})
	if err != nil {
		return fmt.Errorf("calsync: list user links: %w", err)
	}
	conns, err := p.q.ListFamilyGoogleConnections(ctx, familyID)
	if err != nil {
		return fmt.Errorf("calsync: list connections: %w", err)
	}
	var errs []error
	for _, l := range links {
		func() {
			defer p.lockItem(item{kind: l.Kind, familyID: l.FamilyID, itemID: l.ItemID})()
			for _, conn := range conns {
				if conn.UserID == userID || conn.CalendarID != l.CalendarID {
					continue
				}
				other, err := p.q.GetCalendarLink(ctx, db.GetCalendarLinkParams{
					FamilyID: l.FamilyID, UserID: conn.UserID, Kind: l.Kind, ItemID: l.ItemID,
				})
				if err == nil {
					if other.CalendarID == l.CalendarID {
						errs = append(errs, p.dropLink(ctx, l))
						return
					}
					continue
				}
				if !errors.Is(err, pgx.ErrNoRows) {
					errs = append(errs, fmt.Errorf("calsync: get transfer link: %w", err))
					return
				}
				if err := p.q.UpsertCalendarLink(ctx, db.UpsertCalendarLinkParams{
					FamilyID: l.FamilyID, UserID: conn.UserID, Kind: l.Kind, ItemID: l.ItemID,
					CalendarID: l.CalendarID, EventID: l.EventID, Etag: l.Etag,
				}); err != nil {
					errs = append(errs, fmt.Errorf("calsync: transfer link: %w", err))
					return
				}
				errs = append(errs, p.dropLink(ctx, l))
				return
			}
			err = p.unlink(ctx, l)
			if errors.Is(err, gcal.ErrForbidden) {
				p.log.WarnContext(ctx, "abandon calendar event without writer access", slog.String("calendar_id", l.CalendarID), slog.String("event_id", l.EventID))
				err = p.dropLink(ctx, l)
			}
			errs = append(errs, err)
		}()
	}
	return errors.Join(errs...)
}

func (p *Pusher) lockItem(it item) func() {
	key := pgconv.UUIDString(it.familyID) + "|" + it.kind + "|" + pgconv.UUIDString(it.itemID)
	p.mu.Lock()
	m, ok := p.items[key]
	if !ok {
		m = &itemLock{}
		p.items[key] = m
	}
	m.users++
	p.mu.Unlock()
	m.mu.Lock()
	return func() {
		m.mu.Unlock()
		p.mu.Lock()
		m.users--
		if m.users == 0 {
			delete(p.items, key)
		}
		p.mu.Unlock()
	}
}

func (p *Pusher) push(ctx context.Context, it item, restore bool) error {
	var errs []error
	if !it.wanted {
		links, err := p.q.ListCalendarLinksForItem(ctx, db.ListCalendarLinksForItemParams{
			FamilyID: it.familyID, Kind: it.kind, ItemID: it.itemID,
		})
		if err != nil {
			return fmt.Errorf("calsync: list links: %w", err)
		}
		for _, l := range links {
			errs = append(errs, p.unlink(ctx, l))
		}
		return errors.Join(errs...)
	}

	conns, err := p.q.ListFamilyGoogleConnections(ctx, it.familyID)
	if err != nil {
		return fmt.Errorf("calsync: list connections: %w", err)
	}
	for _, conn := range conns {
		errs = append(errs, p.pushTo(ctx, conn, it, restore))
	}
	return errors.Join(errs...)
}

func (p *Pusher) pushTo(ctx context.Context, conn db.GoogleConnection, it item, restore bool) error {
	link, err := p.q.GetCalendarLink(ctx, db.GetCalendarLinkParams{
		FamilyID: it.familyID, UserID: conn.UserID, Kind: it.kind, ItemID: it.itemID,
	})
	hasLink := err == nil
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("calsync: get link: %w", err)
	}

	if hasLink && link.CalendarID != conn.CalendarID {
		if err := p.dropLink(ctx, link); err != nil {
			return err
		}
		hasLink = false
	}

	if !hasLink {
		shared, err := p.q.CalendarLinkedInFamily(ctx, db.CalendarLinkedInFamilyParams{
			FamilyID: it.familyID, Kind: it.kind, ItemID: it.itemID, CalendarID: conn.CalendarID, UserID: conn.UserID,
		})
		if err != nil {
			return fmt.Errorf("calsync: check shared calendar: %w", err)
		}
		if shared {
			return nil
		}
	}

	access, err := p.accessToken(ctx, conn)
	if err != nil {
		return p.fail(ctx, conn, err)
	}

	var saved gcal.Event
	if hasLink {
		cancelled, err := p.cancelledInGoogle(ctx, access, conn, link.EventID)
		if err != nil {
			return p.fail(ctx, conn, err)
		}
		if cancelled {
			return nil
		}
		saved, err = p.google.UpdateEvent(ctx, access, conn.CalendarID, link.EventID, it.event)
		if errors.Is(err, gcal.ErrNotFound) {
			return nil
		}
		if err != nil {
			return p.fail(ctx, conn, err)
		}
	} else {
		it.event.ID = gcal.EventID(pgconv.UUIDString(it.familyID), "", it.kind, pgconv.UUIDString(it.itemID), conn.CalendarID)
		saved, err = p.google.InsertEvent(ctx, access, conn.CalendarID, it.event)
		if errors.Is(err, gcal.ErrConflict) {
			full := conn
			full.SyncToken = ""
			cancelled, cerr := p.cancelledInGoogle(ctx, access, full, it.event.ID)
			if cerr != nil {
				return p.fail(ctx, conn, cerr)
			}
			if cancelled && !restore {
				return nil
			}
			saved, err = p.google.UpdateEvent(ctx, access, conn.CalendarID, it.event.ID, it.event)
		}
		if err != nil {
			return p.fail(ctx, conn, err)
		}
	}

	if err := p.q.UpsertCalendarLink(ctx, db.UpsertCalendarLinkParams{
		FamilyID: it.familyID, UserID: conn.UserID, Kind: it.kind, ItemID: it.itemID,
		CalendarID: conn.CalendarID, EventID: saved.ID, Etag: saved.ETag,
	}); err != nil {
		return fmt.Errorf("calsync: save link: %w", err)
	}
	return nil
}

func (p *Pusher) unlink(ctx context.Context, l db.CalendarLink) error {
	conn, err := p.q.GetGoogleConnection(ctx, db.GetGoogleConnectionParams{FamilyID: l.FamilyID, UserID: l.UserID})
	if errors.Is(err, pgx.ErrNoRows) {
		return p.dropLink(ctx, l)
	}
	if err != nil {
		return fmt.Errorf("calsync: get connection: %w", err)
	}
	access, err := p.accessToken(ctx, conn)
	if err != nil {
		return p.fail(ctx, conn, err)
	}
	err = p.google.DeleteEvent(ctx, access, l.CalendarID, l.EventID)
	if err != nil && !errors.Is(err, gcal.ErrNotFound) {
		return p.fail(ctx, conn, err)
	}
	return p.dropLink(ctx, l)
}

func (p *Pusher) dropLink(ctx context.Context, l db.CalendarLink) error {
	if err := p.q.DeleteCalendarLink(ctx, db.DeleteCalendarLinkParams{
		FamilyID: l.FamilyID, UserID: l.UserID, Kind: l.Kind, ItemID: l.ItemID,
	}); err != nil {
		return fmt.Errorf("calsync: delete link: %w", err)
	}
	return nil
}

func (p *Pusher) cancelledInGoogle(ctx context.Context, access string, conn db.GoogleConnection, eventID string) (bool, error) {
	changes, err := p.google.ListChanges(ctx, access, conn.CalendarID, conn.SyncToken)
	if errors.Is(err, gcal.ErrSyncTokenExpired) {
		changes, err = p.google.ListChanges(ctx, access, conn.CalendarID, "")
	}
	if err != nil {
		return false, err
	}
	for _, e := range changes.Events {
		if e.ID == eventID && e.Cancelled() {
			return true, nil
		}
	}
	return false, nil
}

func (p *Pusher) accessToken(ctx context.Context, conn db.GoogleConnection) (string, error) {
	key := pgconv.UUIDString(conn.FamilyID) + "|" + pgconv.UUIDString(conn.UserID)
	p.mu.Lock()
	tok, ok := p.tokens[key]
	p.mu.Unlock()
	if ok && p.now().Add(time.Minute).Before(tok.Expiry) {
		return tok.AccessToken, nil
	}

	refresh, err := p.box.Open(conn.RefreshToken, crypto.Owner(pgconv.UUIDString(conn.FamilyID), pgconv.UUIDString(conn.UserID)))
	if err != nil {
		return "", fmt.Errorf("calsync: open refresh token: %w", err)
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

func (p *Pusher) fail(ctx context.Context, conn db.GoogleConnection, err error) error {
	text := ErrTextSync
	switch {
	case errors.Is(err, gcal.ErrForbidden):
		text = ErrTextReadOnly
	case errors.Is(err, gcal.ErrUnauthorized), errors.Is(err, gcal.ErrInsufficientScope):
		text = ErrTextReconnect
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

func (p *Pusher) location(ctx context.Context, familyID pgtype.UUID) (*time.Location, error) {
	s, err := p.q.GetFamilySettings(ctx, familyID)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.UTC, nil
	}
	if err != nil {
		return nil, fmt.Errorf("calsync: load family settings: %w", err)
	}
	loc, err := time.LoadLocation(s.Timezone)
	if err != nil {
		p.log.WarnContext(ctx, "unknown family timezone", slog.String("timezone", s.Timezone))
		return time.UTC, nil
	}
	return loc, nil
}

func (p *Pusher) GoogleClient() gcal.Client {
	return p.google
}

func (p *Pusher) Box() *crypto.Box {
	return p.box
}
