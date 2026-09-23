package gcal

import (
	"context"
	"crypto/sha256"
	"encoding/base32"
	"errors"
	"strings"
	"time"
)

var (
	ErrUnauthorized       = errors.New("gcal: the user's google grant is no longer valid")
	ErrClientConfig       = errors.New("gcal: google rejected the oauth client configuration")
	ErrInsufficientScope  = errors.New("gcal: google did not grant calendar and email access")
	ErrForbidden          = errors.New("gcal: no write access to that calendar")
	ErrSyncTokenExpired   = errors.New("gcal: sync token expired, a full resync is needed")
	ErrNotFound           = errors.New("gcal: not found")
	ErrRateLimited        = errors.New("gcal: rate limited")
	ErrMissingRefresh     = errors.New("gcal: google returned no refresh token")
	ErrBadTimeRange       = errors.New("gcal: event end must be after its start")
	ErrUnexpectedResponse = errors.New("gcal: unexpected response from google")
	ErrConflict           = errors.New("gcal: an event with that id already exists")
)

const (
	PropKind   = "fm_kind"
	PropItemID = "fm_item_id"
	PropFamily = "fm_family_id"

	ScopeCalendar       = "https://www.googleapis.com/auth/calendar"
	ScopeCalendarEvents = "https://www.googleapis.com/auth/calendar.events"
	ScopeEmail          = "email"
)

var eventIDEncoding = base32.NewEncoding("0123456789abcdefghijklmnopqrstuv").WithPadding(base32.NoPadding)

func EventID(familyID, userID, kind, itemID, calendarID string) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{familyID, userID, kind, itemID, calendarID}, "\x00")))
	return eventIDEncoding.EncodeToString(sum[:])
}

type Token struct {
	AccessToken  string
	RefreshToken string
	Expiry       time.Time
	Email        string
	Scopes       []string
}

type Calendar struct {
	ID      string
	Name    string
	Primary bool
}

type Event struct {
	ID               string
	ETag             string
	Status           string
	Summary          string
	Description      string
	AllDay           bool
	StartDate        string
	EndDate          string
	Start            time.Time
	End              time.Time
	TimeZone         string
	Recurrence       []string
	RecurringEventID string
	Private          map[string]string
	Updated          time.Time
}

func (e Event) Cancelled() bool { return e.Status == "cancelled" }

func (e Event) validRange() bool {
	if e.AllDay {
		return e.StartDate != "" && e.EndDate > e.StartDate
	}
	return !e.Start.IsZero() && e.End.After(e.Start)
}

type Changes struct {
	Events        []Event
	NextSyncToken string
}

type Client interface {
	Exchange(ctx context.Context, code, verifier, redirectURI string) (Token, error)
	Refresh(ctx context.Context, refreshToken string) (Token, error)
	Revoke(ctx context.Context, token string) error
	ListCalendars(ctx context.Context, accessToken string) ([]Calendar, error)
	InsertEvent(ctx context.Context, accessToken, calendarID string, e Event) (Event, error)
	UpdateEvent(ctx context.Context, accessToken, calendarID, eventID string, e Event) (Event, error)
	DeleteEvent(ctx context.Context, accessToken, calendarID, eventID string) error
	ListChanges(ctx context.Context, accessToken, calendarID, syncToken string) (Changes, error)
}
