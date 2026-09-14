package gcal

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

type fakeEvent struct {
	event Event
	seq   int
}

type grant struct {
	email  string
	scopes []string
}

type Fake struct {
	mu        sync.Mutex
	codes     map[string]grant
	refresh   map[string]grant
	access    map[string]time.Time
	owner     map[string]string
	revoked   []string
	calendars []Calendar
	events    map[string]map[string]*fakeEvent
	seq       int
	nextID    int
	expired   bool
	issued    int
	now       func() time.Time
	Calls     []string
	FailNext  error
}

func NewFake() *Fake {
	return &Fake{
		codes:     map[string]grant{},
		refresh:   map[string]grant{},
		access:    map[string]time.Time{},
		owner:     map[string]string{},
		calendars: []Calendar{{ID: "primary@example.com", Name: "Personal", Primary: true}},
		events:    map[string]map[string]*fakeEvent{},
		now:       time.Now,
	}
}

func (f *Fake) SetNow(now func() time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.now = now
}

func (f *Fake) AddCode(code, verifier, email string) {
	f.AddCodeWithScopes(code, verifier, email, []string{"openid", ScopeEmail, ScopeCalendar})
}

func (f *Fake) AddCodeWithScopes(code, verifier, email string, scopes []string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.codes[code+"|"+verifier] = grant{email: email, scopes: scopes}
}

func (f *Fake) AddCalendar(c Calendar) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calendars = append(f.calendars, c)
}

func (f *Fake) RevokeAll() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.refresh = map[string]grant{}
	f.access = map[string]time.Time{}
	f.owner = map[string]string{}
}

func (f *Fake) Revoked() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string{}, f.revoked...)
}

func (f *Fake) ExpireSyncTokens() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.expired = true
}

func (f *Fake) call(name string) error {
	f.Calls = append(f.Calls, name)
	if f.FailNext != nil {
		err := f.FailNext
		f.FailNext = nil
		return err
	}
	return nil
}

func (f *Fake) authorized(token string) error {
	exp, ok := f.access[token]
	if !ok || !f.now().Before(exp) {
		return ErrUnauthorized
	}
	return nil
}

func (f *Fake) mint(refreshToken string, g grant) Token {
	f.issued++
	access := "access-" + strconv.Itoa(f.issued)
	exp := f.now().Add(time.Hour)
	f.access[access] = exp
	f.owner[access] = refreshToken
	return Token{AccessToken: access, RefreshToken: refreshToken, Expiry: exp, Email: g.email, Scopes: g.scopes}
}

func (f *Fake) Exchange(_ context.Context, code, verifier, _ string) (Token, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.call("Exchange"); err != nil {
		return Token{}, err
	}
	g, ok := f.codes[code+"|"+verifier]
	if !ok {
		return Token{}, ErrUnauthorized
	}
	delete(f.codes, code+"|"+verifier)
	if !grantsCalendar(g.scopes) || g.email == "" {
		f.revoked = append(f.revoked, "code:"+code)
		return Token{}, ErrInsufficientScope
	}
	refresh := "refresh-" + code
	f.refresh[refresh] = g
	return f.mint(refresh, g), nil
}

func (f *Fake) Refresh(_ context.Context, refreshToken string) (Token, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.call("Refresh"); err != nil {
		return Token{}, err
	}
	g, ok := f.refresh[refreshToken]
	if !ok {
		return Token{}, ErrUnauthorized
	}
	t := f.mint(refreshToken, g)
	t.Email = ""
	return t, nil
}

func (f *Fake) Revoke(_ context.Context, token string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.call("Revoke"); err != nil {
		return err
	}
	f.revoked = append(f.revoked, token)
	grantOf := token
	if _, isAccess := f.access[token]; isAccess {
		grantOf = f.owner[token]
	}
	delete(f.refresh, grantOf)
	for access, refresh := range f.owner {
		if refresh == grantOf {
			delete(f.access, access)
			delete(f.owner, access)
		}
	}
	return nil
}

func (f *Fake) ListCalendars(_ context.Context, accessToken string) ([]Calendar, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.call("ListCalendars"); err != nil {
		return nil, err
	}
	if err := f.authorized(accessToken); err != nil {
		return nil, err
	}
	return append([]Calendar{}, f.calendars...), nil
}

func (f *Fake) calendar(id string) map[string]*fakeEvent {
	c, ok := f.events[id]
	if !ok {
		c = map[string]*fakeEvent{}
		f.events[id] = c
	}
	return c
}

func (f *Fake) store(calendarID string, e Event) Event {
	f.seq++
	e.ETag = fmt.Sprintf("\"%d\"", f.seq)
	e.Updated = f.now()
	if e.Status == "" {
		e.Status = "confirmed"
	}
	f.calendar(calendarID)[e.ID] = &fakeEvent{event: e, seq: f.seq}
	return e
}

func (f *Fake) InsertEvent(_ context.Context, accessToken, calendarID string, e Event) (Event, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.call("InsertEvent"); err != nil {
		return Event{}, err
	}
	if err := f.authorized(accessToken); err != nil {
		return Event{}, err
	}
	if !e.validRange() {
		return Event{}, ErrBadTimeRange
	}
	f.nextID++
	e.ID = "evt" + strconv.Itoa(f.nextID)
	e.Status = "confirmed"
	return f.store(calendarID, e), nil
}

func (f *Fake) UpdateEvent(_ context.Context, accessToken, calendarID, eventID string, e Event) (Event, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.call("UpdateEvent"); err != nil {
		return Event{}, err
	}
	if err := f.authorized(accessToken); err != nil {
		return Event{}, err
	}
	if _, ok := f.calendar(calendarID)[eventID]; !ok {
		return Event{}, ErrNotFound
	}
	if !e.validRange() {
		return Event{}, ErrBadTimeRange
	}
	e.ID = eventID
	e.Status = "confirmed"
	return f.store(calendarID, e), nil
}

func (f *Fake) DeleteEvent(_ context.Context, accessToken, calendarID, eventID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.call("DeleteEvent"); err != nil {
		return err
	}
	if err := f.authorized(accessToken); err != nil {
		return err
	}
	f.cancel(calendarID, eventID)
	return nil
}

func (f *Fake) cancel(calendarID, eventID string) {
	cur, ok := f.calendar(calendarID)[eventID]
	if !ok || cur.event.Cancelled() {
		return
	}
	f.store(calendarID, Event{ID: eventID, Status: "cancelled"})
}

func (f *Fake) ListChanges(_ context.Context, accessToken, calendarID, syncToken string) (Changes, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.call("ListChanges"); err != nil {
		return Changes{}, err
	}
	if err := f.authorized(accessToken); err != nil {
		return Changes{}, err
	}
	since := 0
	if syncToken != "" {
		if f.expired {
			f.expired = false
			return Changes{}, ErrSyncTokenExpired
		}
		n, err := strconv.Atoi(strings.TrimPrefix(syncToken, "sync-"))
		if err != nil {
			return Changes{}, ErrSyncTokenExpired
		}
		since = n
	}
	var out Changes
	for _, fe := range f.calendar(calendarID) {
		if fe.seq > since {
			out.Events = append(out.Events, fe.event)
		}
	}
	out.NextSyncToken = "sync-" + strconv.Itoa(f.seq)
	return out, nil
}

func (f *Fake) EditInGoogle(calendarID, eventID string, edit func(*Event)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	cur, ok := f.calendar(calendarID)[eventID]
	if !ok || cur.event.Cancelled() {
		return
	}
	e := cur.event
	edit(&e)
	f.store(calendarID, e)
}

func (f *Fake) DeleteInGoogle(calendarID, eventID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cancel(calendarID, eventID)
}

func (f *Fake) Event(calendarID, eventID string) (Event, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	cur, ok := f.calendar(calendarID)[eventID]
	if !ok {
		return Event{}, false
	}
	return cur.event, true
}

func (f *Fake) Live(calendarID string) []Event {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Event
	for _, fe := range f.calendar(calendarID) {
		if !fe.event.Cancelled() {
			out = append(out, fe.event)
		}
	}
	return out
}

var _ Client = (*Fake)(nil)
var _ Client = (*HTTPClient)(nil)
