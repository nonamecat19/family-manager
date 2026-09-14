package gcal

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

type google struct {
	t        *testing.T
	srv      *httptest.Server
	lastForm url.Values
	lastBody map[string]any
	lastPath string
	lastAuth string
}

func newGoogle(t *testing.T, handle func(g *google, w http.ResponseWriter, r *http.Request)) (*google, *HTTPClient) {
	g := &google{t: t}
	g.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.lastPath = r.URL.EscapedPath()
		g.lastAuth = r.Header.Get("Authorization")
		if r.Header.Get("Content-Type") == "application/x-www-form-urlencoded" {
			_ = r.ParseForm()
			g.lastForm = r.PostForm
		} else if r.Body != nil {
			raw, _ := io.ReadAll(r.Body)
			g.lastBody = nil
			_ = json.Unmarshal(raw, &g.lastBody)
		}
		handle(g, w, r)
	}))
	t.Cleanup(g.srv.Close)
	c := NewHTTP(HTTPOptions{
		ClientID: "cid", ClientSecret: "csecret",
		TokenURL: g.srv.URL + "/token", APIBase: g.srv.URL + "/calendar/v3",
		Now: func() time.Time { return time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC) },
	})
	return g, c
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func idToken(email string) string {
	payload, _ := json.Marshal(map[string]string{"email": email})
	return "h." + base64.RawURLEncoding.EncodeToString(payload) + ".s"
}

func TestExchangeSendsPKCEAndReadsEmail(t *testing.T) {
	g, c := newGoogle(t, func(_ *google, w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, 200, map[string]any{"access_token": "at", "refresh_token": "rt", "expires_in": 3600,
			"id_token": idToken("ann@example.com"), "scope": "openid " + ScopeEmail + " " + ScopeCalendar})
	})
	tok, err := c.Exchange(context.Background(), "the-code", "the-verifier", "fmtasks:/oauthredirect")
	if err != nil {
		t.Fatal(err)
	}
	f := g.lastForm
	if f.Get("grant_type") != "authorization_code" || f.Get("code") != "the-code" || f.Get("code_verifier") != "the-verifier" ||
		f.Get("redirect_uri") != "fmtasks:/oauthredirect" || f.Get("client_id") != "cid" || f.Get("client_secret") != "csecret" {
		t.Fatalf("form: %v", f)
	}
	if tok.RefreshToken != "rt" || tok.AccessToken != "at" || tok.Email != "ann@example.com" ||
		!tok.Expiry.Equal(time.Date(2026, 10, 5, 13, 0, 0, 0, time.UTC)) {
		t.Fatalf("token: %+v", tok)
	}
}

func TestExchangeWithoutRefreshTokenFails(t *testing.T) {
	_, c := newGoogle(t, func(_ *google, w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, 200, map[string]any{"access_token": "at", "expires_in": 3600,
			"id_token": idToken("a@x"), "scope": ScopeCalendar + " email"})
	})
	if _, err := c.Exchange(context.Background(), "c", "v", "r"); !errors.Is(err, ErrMissingRefresh) {
		t.Fatalf("err = %v", err)
	}
}

func TestRefreshInvalidGrantIsUnauthorized(t *testing.T) {
	_, c := newGoogle(t, func(_ *google, w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, 400, map[string]any{"error": "invalid_grant"})
	})
	if _, err := c.Refresh(context.Background(), "rt"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("err = %v", err)
	}
}

func TestRefreshKeepsTheRefreshToken(t *testing.T) {
	g, c := newGoogle(t, func(_ *google, w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, 200, map[string]any{"access_token": "at2", "expires_in": 3600})
	})
	tok, err := c.Refresh(context.Background(), "rt")
	if err != nil || tok.RefreshToken != "rt" || tok.AccessToken != "at2" || g.lastForm.Get("grant_type") != "refresh_token" {
		t.Fatalf("%+v %v %v", tok, err, g.lastForm)
	}
}

func TestListCalendarsFollowsPages(t *testing.T) {
	g, c := newGoogle(t, func(_ *google, w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("pageToken") == "" {
			writeJSON(w, 200, map[string]any{"items": []any{map[string]any{"id": "a@x", "summary": "A", "primary": true}}, "nextPageToken": "p2"})
			return
		}
		writeJSON(w, 200, map[string]any{"items": []any{map[string]any{"id": "family#b@group", "summary": "Family"}}})
	})
	cals, err := c.ListCalendars(context.Background(), "at")
	if err != nil || len(cals) != 2 || !cals[0].Primary || cals[1].ID != "family#b@group" {
		t.Fatalf("%+v %v", cals, err)
	}
	if g.lastAuth != "Bearer at" {
		t.Fatalf("auth header %q", g.lastAuth)
	}
}

func TestInsertAllDayEventEscapesCalendarID(t *testing.T) {
	g, c := newGoogle(t, func(_ *google, w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, 200, map[string]any{"id": "e1", "etag": "\"1\"", "status": "confirmed", "summary": "Pay rent",
			"start": map[string]any{"date": "2026-10-05"}, "end": map[string]any{"date": "2026-10-06"},
			"extendedProperties": map[string]any{"private": map[string]any{PropKind: "task", PropItemID: "t1"}}})
	})
	e, err := c.InsertEvent(context.Background(), "at", "family#b@group.calendar.google.com", Event{
		Summary: "Pay rent", AllDay: true, StartDate: "2026-10-05", EndDate: "2026-10-06",
		Private: map[string]string{PropKind: "task", PropItemID: "t1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if g.lastPath != "/calendar/v3/calendars/family%23b@group.calendar.google.com/events" {
		t.Fatalf("path %s", g.lastPath)
	}
	start, ok := g.lastBody["start"].(map[string]any)
	if !ok {
		t.Fatalf("body start %v", g.lastBody["start"])
	}
	if start["date"] != "2026-10-05" || start["dateTime"] != nil {
		t.Fatalf("body start %v", start)
	}
	if !e.AllDay || e.ID != "e1" || e.Private[PropItemID] != "t1" {
		t.Fatalf("event %+v", e)
	}
}

func TestUpdateSwitchingToAllDaySendsExplicitNulls(t *testing.T) {
	g, c := newGoogle(t, func(_ *google, w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method %s, want PUT", r.Method)
		}
		writeJSON(w, 200, map[string]any{"id": "e1", "status": "confirmed"})
	})
	_, err := c.UpdateEvent(context.Background(), "at", "primary", "e1", Event{Summary: "x", AllDay: true, StartDate: "2026-10-05", EndDate: "2026-10-06"})
	if err != nil {
		t.Fatal(err)
	}
	st, ok := g.lastBody["start"].(map[string]any)
	if !ok {
		t.Fatalf("body start %v", g.lastBody["start"])
	}
	if v, ok := st["dateTime"]; !ok || v != nil {
		t.Fatalf("dateTime must be sent as explicit null, got %v (present %v)", v, ok)
	}
	if g.lastBody["status"] != "confirmed" {
		t.Fatalf("an update must restore a cancelled event: %v", g.lastBody["status"])
	}
}

func TestUpdateTimedEventSendsRFC3339AndZone(t *testing.T) {
	g, c := newGoogle(t, func(_ *google, w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, 200, map[string]any{"id": "e1"})
	})
	kyiv, _ := time.LoadLocation("Europe/Kyiv")
	start := time.Date(2026, 10, 5, 18, 0, 0, 0, kyiv)
	_, err := c.UpdateEvent(context.Background(), "at", "primary", "e1", Event{Summary: "Call", Start: start, End: start.Add(30 * time.Minute), TimeZone: "Europe/Kyiv"})
	if err != nil {
		t.Fatal(err)
	}
	st, ok := g.lastBody["start"].(map[string]any)
	if !ok {
		t.Fatalf("body start %v", g.lastBody["start"])
	}
	if st["dateTime"] != "2026-10-05T18:00:00+03:00" || st["timeZone"] != "Europe/Kyiv" || st["date"] != nil {
		t.Fatalf("start %v", st)
	}
}

func TestBadTimeRangeIsRejectedBeforeCallingGoogle(t *testing.T) {
	g, c := newGoogle(t, func(_ *google, w http.ResponseWriter, _ *http.Request) { writeJSON(w, 200, map[string]any{}) })
	_, err := c.InsertEvent(context.Background(), "at", "primary", Event{AllDay: true, StartDate: "2026-10-05", EndDate: "2026-10-05"})
	if !errors.Is(err, ErrBadTimeRange) || g.lastPath != "" {
		t.Fatalf("err %v, path %q", err, g.lastPath)
	}
}

func TestUpdateOfGoneEventIsNotFound(t *testing.T) {
	_, c := newGoogle(t, func(_ *google, w http.ResponseWriter, _ *http.Request) { w.WriteHeader(410) })
	_, err := c.UpdateEvent(context.Background(), "at", "primary", "e1", Event{AllDay: true, StartDate: "2026-10-05", EndDate: "2026-10-06"})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("410 outside events.list means gone: %v", err)
	}
}

func TestExchangeRequiresCalendarScopeAndEmail(t *testing.T) {
	for name, scope := range map[string]string{"no calendar": "openid email", "no email": ScopeCalendar} {
		t.Run(name, func(t *testing.T) {
			id := idToken("a@x")
			if name == "no email" {
				id = ""
			}
			revoked := false
			_, c := newGoogle(t, func(_ *google, w http.ResponseWriter, r *http.Request) {
				if strings.HasSuffix(r.URL.Path, "/revoke") {
					revoked = true
					return
				}
				writeJSON(w, 200, map[string]any{"access_token": "at", "refresh_token": "rt", "expires_in": 3600, "id_token": id, "scope": scope})
			})
			c.opts.RevokeURL = strings.Replace(c.opts.TokenURL, "/token", "/revoke", 1)
			if _, err := c.Exchange(context.Background(), "c", "v", "r"); !errors.Is(err, ErrInsufficientScope) {
				t.Fatalf("err = %v", err)
			}
			if !revoked {
				t.Fatal("a grant without the needed scopes must be revoked")
			}
		})
	}
}

func TestInvalidClientIsAConfigErrorNotARevokedGrant(t *testing.T) {
	_, c := newGoogle(t, func(_ *google, w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, 401, map[string]any{"error": "invalid_client"})
	})
	_, err := c.Refresh(context.Background(), "rt")
	if !errors.Is(err, ErrClientConfig) || errors.Is(err, ErrUnauthorized) {
		t.Fatalf("err = %v", err)
	}
}

func TestEmptyAccessTokenIsUnexpected(t *testing.T) {
	_, c := newGoogle(t, func(_ *google, w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, 200, map[string]any{"expires_in": 3600})
	})
	if _, err := c.Refresh(context.Background(), "rt"); !errors.Is(err, ErrUnexpectedResponse) {
		t.Fatalf("err = %v", err)
	}
}

func TestClientSecretOmittedWhenEmpty(t *testing.T) {
	g, c := newGoogle(t, func(_ *google, w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, 200, map[string]any{"access_token": "at", "expires_in": 3600})
	})
	c.opts.ClientSecret = ""
	_, _ = c.Refresh(context.Background(), "rt")
	if _, present := g.lastForm["client_secret"]; present {
		t.Fatal("client_secret must not be sent when unset")
	}
}

func TestRevokePostsTheToken(t *testing.T) {
	g, c := newGoogle(t, func(_ *google, w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) })
	c.opts.RevokeURL = c.opts.TokenURL
	if err := c.Revoke(context.Background(), "rt"); err != nil || g.lastForm.Get("token") != "rt" {
		t.Fatalf("%v %v", err, g.lastForm)
	}
}

func TestForbiddenReasons(t *testing.T) {
	cases := map[string]error{"rateLimitExceeded": ErrRateLimited, "userRateLimitExceeded": ErrRateLimited,
		"dailyLimitExceeded": ErrRateLimited, "insufficientPermissions": ErrInsufficientScope, "forbidden": ErrForbidden}
	for reason, want := range cases {
		_, c := newGoogle(t, func(_ *google, w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, 403, map[string]any{"error": map[string]any{"errors": []any{map[string]any{"reason": reason}}}})
		})
		if _, err := c.ListCalendars(context.Background(), "at"); !errors.Is(err, want) {
			t.Errorf("%s: err = %v, want %v", reason, err, want)
		}
	}
}

func TestDeleteToleratesMissingEvent(t *testing.T) {
	for _, status := range []int{404, 410} {
		_, c := newGoogle(t, func(_ *google, w http.ResponseWriter, _ *http.Request) { w.WriteHeader(status) })
		if err := c.DeleteEvent(context.Background(), "at", "primary", "gone"); err != nil {
			t.Fatalf("status %d: %v", status, err)
		}
	}
}

func TestListChangesPagesAndReturnsSyncToken(t *testing.T) {
	g, c := newGoogle(t, func(_ *google, w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("showDeleted") != "true" || q.Get("syncToken") != "s1" {
			t.Errorf("query %v", q)
		}
		if q.Get("pageToken") == "" {
			writeJSON(w, 200, map[string]any{"items": []any{map[string]any{"id": "a", "status": "cancelled"}}, "nextPageToken": "p2"})
			return
		}
		writeJSON(w, 200, map[string]any{"items": []any{map[string]any{"id": "b", "summary": "B", "updated": "2026-10-05T10:00:00Z"}}, "nextSyncToken": "s2"})
	})
	ch, err := c.ListChanges(context.Background(), "at", "primary", "s1")
	if err != nil {
		t.Fatal(err)
	}
	if len(ch.Events) != 2 || !ch.Events[0].Cancelled() || ch.NextSyncToken != "s2" || ch.Events[1].Updated.IsZero() {
		t.Fatalf("%+v", ch)
	}
	_ = g
}

func TestStatusMapping(t *testing.T) {
	cases := map[int]error{401: ErrUnauthorized, 404: ErrNotFound, 410: ErrSyncTokenExpired, 429: ErrRateLimited, 500: ErrUnexpectedResponse}
	for status, want := range cases {
		_, c := newGoogle(t, func(_ *google, w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(strings.Repeat("x", 10)))
		})
		if _, err := c.ListChanges(context.Background(), "at", "primary", "s"); !errors.Is(err, want) {
			t.Errorf("status %d: err = %v, want %v", status, err, want)
		}
	}
}

func TestFakeMatchesGoogleSemantics(t *testing.T) {
	ctx := context.Background()
	clock := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	f := NewFake()
	f.SetNow(func() time.Time { return clock })
	f.AddCode("code", "ver", "ann@example.com")
	tok, err := f.Exchange(ctx, "code", "ver", "r")
	if err != nil {
		t.Fatal(err)
	}
	cal := "primary@example.com"
	day := Event{Summary: "x", AllDay: true, StartDate: "2026-10-05", EndDate: "2026-10-06", Private: map[string]string{PropItemID: "t1"}}
	full, _ := f.ListChanges(ctx, tok.AccessToken, cal, "")
	a, _ := f.InsertEvent(ctx, tok.AccessToken, cal, day)
	b, _ := f.InsertEvent(ctx, tok.AccessToken, cal, day)
	f.EditInGoogle(cal, a.ID, func(ev *Event) { ev.Summary = "edited" })
	f.DeleteInGoogle(cal, b.ID)
	next, _ := f.ListChanges(ctx, tok.AccessToken, cal, full.NextSyncToken)
	var gone Event
	for _, e := range next.Events {
		if e.ID == b.ID {
			gone = e
		}
	}
	if !gone.Cancelled() || gone.Summary != "" || gone.Private != nil {
		t.Fatalf("a cancelled event carries only its id and status: %+v", gone)
	}
	resync, _ := f.ListChanges(ctx, tok.AccessToken, cal, "")
	sawCancelled := false
	for _, e := range resync.Events {
		sawCancelled = sawCancelled || e.Cancelled()
	}
	if !sawCancelled {
		t.Fatal("a full sync with showDeleted returns cancelled events too")
	}
	if _, err := f.InsertEvent(ctx, tok.AccessToken, cal, Event{AllDay: true, StartDate: "2026-10-05", EndDate: "2026-10-05"}); !errors.Is(err, ErrBadTimeRange) {
		t.Fatalf("empty range: %v", err)
	}
	restored, err := f.UpdateEvent(ctx, tok.AccessToken, cal, b.ID, day)
	if err != nil || restored.Cancelled() {
		t.Fatalf("an update restores a deleted event: %+v %v", restored, err)
	}
	f.ExpireSyncTokens()
	if _, err := f.ListChanges(ctx, tok.AccessToken, cal, next.NextSyncToken); !errors.Is(err, ErrSyncTokenExpired) {
		t.Fatalf("expected expiry, got %v", err)
	}
	clock = clock.Add(2 * time.Hour)
	if _, err := f.ListCalendars(ctx, tok.AccessToken); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("access tokens expire after an hour: %v", err)
	}
	fresh, err := f.Refresh(ctx, tok.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.ListCalendars(ctx, fresh.AccessToken); err != nil {
		t.Fatal(err)
	}
	_ = f.Revoke(ctx, fresh.AccessToken)
	if _, err := f.Refresh(ctx, tok.RefreshToken); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("revoking an access token revokes the whole grant: %v", err)
	}
	if _, err := f.ListCalendars(ctx, fresh.AccessToken); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("revoked access token still works: %v", err)
	}
	f.AddCodeWithScopes("narrow", "v", "a@x", []string{"openid", "email"})
	if _, err := f.Exchange(ctx, "narrow", "v", "r"); !errors.Is(err, ErrInsufficientScope) {
		t.Fatalf("missing calendar scope: %v", err)
	}
}

func TestEventIDIsStableAndGoogleSafe(t *testing.T) {
	id := EventID("fam1", "user1", "task", "item1", "primary@example.com")
	if id != EventID("fam1", "user1", "task", "item1", "primary@example.com") {
		t.Fatal("the same inputs must give the same id")
	}
	if len(id) < 5 || len(id) > 1024 {
		t.Fatalf("length %d outside 5-1024", len(id))
	}
	for _, r := range id {
		if (r < '0' || r > '9') && (r < 'a' || r > 'v') {
			t.Fatalf("id %q has %q outside a-v0-9", id, r)
		}
	}
	others := []string{
		EventID("fam2", "user1", "task", "item1", "primary@example.com"),
		EventID("fam1", "user2", "task", "item1", "primary@example.com"),
		EventID("fam1", "user1", "birthday", "item1", "primary@example.com"),
		EventID("fam1", "user1", "task", "item2", "primary@example.com"),
		EventID("fam1", "user1", "task", "item1", "family@group.calendar.google.com"),
		EventID("fam1user1", "", "task", "item1", "primary@example.com"),
	}
	seen := map[string]bool{id: true}
	for i, o := range others {
		if seen[o] {
			t.Fatalf("variant %d collides: %s", i, o)
		}
		seen[o] = true
	}
}

func TestInsertSendsClientEventID(t *testing.T) {
	g, c := newGoogle(t, func(_ *google, w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, 200, map[string]any{"id": "abc123", "status": "confirmed"})
	})
	id := EventID("f", "u", "task", "t1", "primary")
	_, err := c.InsertEvent(context.Background(), "at", "primary", Event{ID: id, AllDay: true, StartDate: "2026-10-05", EndDate: "2026-10-06"})
	if err != nil {
		t.Fatal(err)
	}
	if g.lastBody["id"] != id {
		t.Fatalf("body id %v, want %s", g.lastBody["id"], id)
	}
	_, err = c.InsertEvent(context.Background(), "at", "primary", Event{AllDay: true, StartDate: "2026-10-05", EndDate: "2026-10-06"})
	if err != nil {
		t.Fatal(err)
	}
	if _, present := g.lastBody["id"]; present {
		t.Fatalf("id must be omitted when unset: %v", g.lastBody)
	}
}

func TestUpdateDoesNotSendEventID(t *testing.T) {
	g, c := newGoogle(t, func(_ *google, w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, 200, map[string]any{"id": "e1"})
	})
	_, err := c.UpdateEvent(context.Background(), "at", "primary", "e1", Event{ID: "other", AllDay: true, StartDate: "2026-10-05", EndDate: "2026-10-06"})
	if err != nil {
		t.Fatal(err)
	}
	if _, present := g.lastBody["id"]; present {
		t.Fatalf("update body must not carry an id: %v", g.lastBody)
	}
}

func TestInsertConflictIsErrConflict(t *testing.T) {
	_, c := newGoogle(t, func(_ *google, w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, 409, map[string]any{"error": map[string]any{"code": 409, "errors": []any{map[string]any{"reason": "duplicate"}}}})
	})
	_, err := c.InsertEvent(context.Background(), "at", "primary", Event{ID: EventID("f", "u", "task", "t1", "primary"), AllDay: true, StartDate: "2026-10-05", EndDate: "2026-10-06"})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("err = %v", err)
	}
}

func TestFakeClientEventIDsAndConflicts(t *testing.T) {
	ctx := context.Background()
	f := NewFake()
	f.AddCode("code", "ver", "ann@example.com")
	tok, err := f.Exchange(ctx, "code", "ver", "r")
	if err != nil {
		t.Fatal(err)
	}
	cal := "primary@example.com"
	day := Event{Summary: "x", AllDay: true, StartDate: "2026-10-05", EndDate: "2026-10-06"}
	auto, err := f.InsertEvent(ctx, tok.AccessToken, cal, day)
	if err != nil || auto.ID != "evt1" {
		t.Fatalf("without an id the fake assigns evtN: %+v %v", auto, err)
	}
	withID := day
	withID.ID = EventID("f", "u", "task", "t1", cal)
	got, err := f.InsertEvent(ctx, tok.AccessToken, cal, withID)
	if err != nil || got.ID != withID.ID {
		t.Fatalf("client id not used: %+v %v", got, err)
	}
	if _, err := f.InsertEvent(ctx, tok.AccessToken, cal, withID); !errors.Is(err, ErrConflict) {
		t.Fatalf("live duplicate: %v", err)
	}
	f.DeleteInGoogle(cal, withID.ID)
	if _, err := f.InsertEvent(ctx, tok.AccessToken, cal, withID); !errors.Is(err, ErrConflict) {
		t.Fatalf("cancelled duplicate: %v", err)
	}
	other := withID
	other.ID = EventID("f", "u", "task", "t1", "family@group")
	if _, err := f.InsertEvent(ctx, "", "family@group", other); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("auth still checked: %v", err)
	}
	if _, err := f.InsertEvent(ctx, tok.AccessToken, "family@group", withID); err != nil {
		t.Fatalf("ids are scoped per calendar: %v", err)
	}
	next, err := f.InsertEvent(ctx, tok.AccessToken, cal, day)
	if err != nil || next.ID != "evt2" {
		t.Fatalf("evtN sequence must not be consumed by client ids: %+v %v", next, err)
	}
}
