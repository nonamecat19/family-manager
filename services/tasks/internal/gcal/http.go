package gcal

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"
)

const (
	DefaultTokenURL  = "https://oauth2.googleapis.com/token"
	DefaultRevokeURL = "https://oauth2.googleapis.com/revoke"
	DefaultAPIBase   = "https://www.googleapis.com/calendar/v3"
	maxBody          = 4 << 20
)

type HTTPOptions struct {
	ClientID     string
	ClientSecret string
	TokenURL     string
	RevokeURL    string
	APIBase      string
	HTTP         *http.Client
	Now          func() time.Time
}

type HTTPClient struct {
	opts HTTPOptions
}

func NewHTTP(opts HTTPOptions) *HTTPClient {
	if opts.TokenURL == "" {
		opts.TokenURL = DefaultTokenURL
	}
	if opts.RevokeURL == "" {
		opts.RevokeURL = DefaultRevokeURL
	}
	if opts.APIBase == "" {
		opts.APIBase = DefaultAPIBase
	}
	opts.APIBase = strings.TrimSuffix(opts.APIBase, "/")
	if opts.HTTP == nil {
		opts.HTTP = &http.Client{Timeout: 20 * time.Second}
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	return &HTTPClient{opts: opts}
}

func transportError(op string, err error) error {
	var ue *url.Error
	if errors.As(err, &ue) {
		err = ue.Err
	}
	return fmt.Errorf("gcal: %s: %w", op, err)
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	IDToken      string `json:"id_token"`
	Scope        string `json:"scope"`
	Error        string `json:"error"`
}

func (c *HTTPClient) post(ctx context.Context, endpoint string, form url.Values) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := c.opts.HTTP.Do(req)
	if err != nil {
		return nil, transportError("oauth request", err)
	}
	return res, nil
}

func (c *HTTPClient) token(ctx context.Context, form url.Values) (Token, error) {
	form.Set("client_id", c.opts.ClientID)
	if c.opts.ClientSecret != "" {
		form.Set("client_secret", c.opts.ClientSecret)
	}
	res, err := c.post(ctx, c.opts.TokenURL, form)
	if err != nil {
		return Token{}, err
	}
	defer func() { _ = res.Body.Close() }()
	var body tokenResponse
	if err := json.NewDecoder(io.LimitReader(res.Body, maxBody)).Decode(&body); err != nil {
		return Token{}, fmt.Errorf("%w: token status %d", ErrUnexpectedResponse, res.StatusCode)
	}
	if res.StatusCode != http.StatusOK {
		switch body.Error {
		case "invalid_grant":
			return Token{}, ErrUnauthorized
		case "invalid_client", "unauthorized_client":
			return Token{}, ErrClientConfig
		}
		return Token{}, fmt.Errorf("%w: token status %d %s", ErrUnexpectedResponse, res.StatusCode, body.Error)
	}
	if body.AccessToken == "" {
		return Token{}, fmt.Errorf("%w: token response without an access token", ErrUnexpectedResponse)
	}
	return Token{
		AccessToken:  body.AccessToken,
		RefreshToken: body.RefreshToken,
		Expiry:       c.opts.Now().Add(time.Duration(body.ExpiresIn) * time.Second),
		Email:        emailFromIDToken(body.IDToken),
		Scopes:       strings.Fields(body.Scope),
	}, nil
}

func emailFromIDToken(idToken string) string {
	parts := strings.Split(idToken, ".")
	if len(parts) < 2 {
		return ""
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var claims struct {
		Email string `json:"email"`
	}
	if json.Unmarshal(raw, &claims) != nil {
		return ""
	}
	return claims.Email
}

func grantsCalendar(scopes []string) bool {
	return slices.Contains(scopes, ScopeCalendar) || slices.Contains(scopes, ScopeCalendarEvents)
}

func (c *HTTPClient) Exchange(ctx context.Context, code, verifier, redirectURI string) (Token, error) {
	t, err := c.token(ctx, url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"code_verifier": {verifier},
		"redirect_uri":  {redirectURI},
	})
	if err != nil {
		return Token{}, err
	}
	if !grantsCalendar(t.Scopes) || t.Email == "" {
		_ = c.Revoke(ctx, t.AccessToken)
		return Token{}, ErrInsufficientScope
	}
	if t.RefreshToken == "" {
		return Token{}, ErrMissingRefresh
	}
	return t, nil
}

func (c *HTTPClient) Refresh(ctx context.Context, refreshToken string) (Token, error) {
	t, err := c.token(ctx, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	})
	if err != nil {
		return Token{}, err
	}
	if t.RefreshToken == "" {
		t.RefreshToken = refreshToken
	}
	return t, nil
}

func (c *HTTPClient) Revoke(ctx context.Context, token string) error {
	res, err := c.post(ctx, c.opts.RevokeURL, url.Values{"token": {token}})
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, maxBody))
	if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusBadRequest {
		return nil
	}
	return fmt.Errorf("%w: revoke status %d", ErrUnexpectedResponse, res.StatusCode)
}

type apiError struct {
	Error struct {
		Errors []struct {
			Reason string `json:"reason"`
		} `json:"errors"`
		Status string `json:"status"`
	} `json:"error"`
}

func forbidden(body []byte) error {
	var e apiError
	_ = json.Unmarshal(body, &e)
	for _, r := range e.Error.Errors {
		switch r.Reason {
		case "rateLimitExceeded", "userRateLimitExceeded", "quotaExceeded", "dailyLimitExceeded":
			return ErrRateLimited
		case "insufficientPermissions":
			return ErrInsufficientScope
		case "forbidden", "requiredAccessLevel", "forbiddenForNonOrganizer":
			return ErrForbidden
		}
	}
	return fmt.Errorf("%w: forbidden", ErrUnexpectedResponse)
}

func (c *HTTPClient) do(ctx context.Context, method, path, accessToken string, query url.Values, in, out any) error {
	u := c.opts.APIBase + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	var body io.Reader
	if in != nil {
		raw, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.opts.HTTP.Do(req)
	if err != nil {
		return transportError(method+" "+path, err)
	}
	defer func() { _ = res.Body.Close() }()
	switch {
	case res.StatusCode == http.StatusUnauthorized:
		return ErrUnauthorized
	case res.StatusCode == http.StatusForbidden:
		raw, _ := io.ReadAll(io.LimitReader(res.Body, 64<<10))
		return forbidden(raw)
	case res.StatusCode == http.StatusNotFound:
		return ErrNotFound
	case res.StatusCode == http.StatusGone:
		if method == http.MethodGet && strings.HasSuffix(path, "/events") {
			return ErrSyncTokenExpired
		}
		return ErrNotFound
	case res.StatusCode == http.StatusConflict:
		return ErrConflict
	case res.StatusCode == http.StatusTooManyRequests:
		return ErrRateLimited
	case res.StatusCode >= 300:
		return fmt.Errorf("%w: %s %s -> %d", ErrUnexpectedResponse, method, path, res.StatusCode)
	}
	if out == nil {
		_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, maxBody))
		return nil
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, maxBody)).Decode(out); err != nil {
		return fmt.Errorf("%w: decode %s", ErrUnexpectedResponse, path)
	}
	return nil
}

func calendarPath(calendarID string) string {
	return "/calendars/" + url.PathEscape(calendarID)
}

func (c *HTTPClient) ListCalendars(ctx context.Context, accessToken string) ([]Calendar, error) {
	var out []Calendar
	page := ""
	for {
		q := url.Values{"minAccessRole": {"writer"}}
		if page != "" {
			q.Set("pageToken", page)
		}
		var body struct {
			Items []struct {
				ID      string `json:"id"`
				Summary string `json:"summary"`
				Primary bool   `json:"primary"`
			} `json:"items"`
			NextPageToken string `json:"nextPageToken"`
		}
		if err := c.do(ctx, http.MethodGet, "/users/me/calendarList", accessToken, q, nil, &body); err != nil {
			return nil, err
		}
		for _, it := range body.Items {
			out = append(out, Calendar{ID: it.ID, Name: it.Summary, Primary: it.Primary})
		}
		if body.NextPageToken == "" {
			return out, nil
		}
		page = body.NextPageToken
	}
}

type wireTime struct {
	Date     *string `json:"date"`
	DateTime *string `json:"dateTime"`
	TimeZone *string `json:"timeZone"`
}

type wireEvent struct {
	ID                 string     `json:"id,omitempty"`
	ETag               string     `json:"etag,omitempty"`
	Status             string     `json:"status,omitempty"`
	Summary            string     `json:"summary,omitempty"`
	Description        string     `json:"description,omitempty"`
	Start              *wireTime  `json:"start,omitempty"`
	End                *wireTime  `json:"end,omitempty"`
	Recurrence         []string   `json:"recurrence,omitempty"`
	RecurringEventID   string     `json:"recurringEventId,omitempty"`
	Updated            string     `json:"updated,omitempty"`
	ExtendedProperties *wireProps `json:"extendedProperties,omitempty"`
}

type wireProps struct {
	Private map[string]string `json:"private,omitempty"`
}

func str(s string) *string { return &s }

func toWire(e Event) wireEvent {
	w := wireEvent{Status: "confirmed", Summary: e.Summary, Description: e.Description, Recurrence: e.Recurrence}
	if e.AllDay {
		w.Start = &wireTime{Date: str(e.StartDate)}
		w.End = &wireTime{Date: str(e.EndDate)}
	} else {
		var zone *string
		if e.TimeZone != "" {
			zone = str(e.TimeZone)
		}
		w.Start = &wireTime{DateTime: str(e.Start.Format(time.RFC3339)), TimeZone: zone}
		w.End = &wireTime{DateTime: str(e.End.Format(time.RFC3339)), TimeZone: zone}
	}
	if len(e.Private) > 0 {
		w.ExtendedProperties = &wireProps{Private: e.Private}
	}
	return w
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func fromWire(w wireEvent) Event {
	e := Event{ID: w.ID, ETag: w.ETag, Status: w.Status, Summary: w.Summary, Description: w.Description,
		Recurrence: w.Recurrence, RecurringEventID: w.RecurringEventID}
	if w.Start != nil {
		if d := deref(w.Start.Date); d != "" {
			e.AllDay = true
			e.StartDate = d
		} else {
			e.Start, _ = time.Parse(time.RFC3339, deref(w.Start.DateTime))
			e.TimeZone = deref(w.Start.TimeZone)
		}
	}
	if w.End != nil {
		if d := deref(w.End.Date); d != "" {
			e.EndDate = d
		} else {
			e.End, _ = time.Parse(time.RFC3339, deref(w.End.DateTime))
		}
	}
	if w.Updated != "" {
		e.Updated, _ = time.Parse(time.RFC3339, w.Updated)
	}
	if w.ExtendedProperties != nil {
		e.Private = w.ExtendedProperties.Private
	}
	return e
}

func (c *HTTPClient) InsertEvent(ctx context.Context, accessToken, calendarID string, e Event) (Event, error) {
	if !e.validRange() {
		return Event{}, ErrBadTimeRange
	}
	w := toWire(e)
	w.ID = e.ID
	var out wireEvent
	if err := c.do(ctx, http.MethodPost, calendarPath(calendarID)+"/events", accessToken, nil, w, &out); err != nil {
		return Event{}, err
	}
	return fromWire(out), nil
}

func (c *HTTPClient) UpdateEvent(ctx context.Context, accessToken, calendarID, eventID string, e Event) (Event, error) {
	if !e.validRange() {
		return Event{}, ErrBadTimeRange
	}
	var out wireEvent
	path := calendarPath(calendarID) + "/events/" + url.PathEscape(eventID)
	if err := c.do(ctx, http.MethodPut, path, accessToken, nil, toWire(e), &out); err != nil {
		return Event{}, err
	}
	return fromWire(out), nil
}

func (c *HTTPClient) DeleteEvent(ctx context.Context, accessToken, calendarID, eventID string) error {
	path := calendarPath(calendarID) + "/events/" + url.PathEscape(eventID)
	err := c.do(ctx, http.MethodDelete, path, accessToken, nil, nil, nil)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	return err
}

func (c *HTTPClient) ListChanges(ctx context.Context, accessToken, calendarID, syncToken string) (Changes, error) {
	var out Changes
	page := ""
	for {
		q := url.Values{"showDeleted": {"true"}, "maxResults": {"250"}}
		if syncToken != "" {
			q.Set("syncToken", syncToken)
		}
		if page != "" {
			q.Set("pageToken", page)
		}
		var body struct {
			Items         []wireEvent `json:"items"`
			NextPageToken string      `json:"nextPageToken"`
			NextSyncToken string      `json:"nextSyncToken"`
		}
		if err := c.do(ctx, http.MethodGet, calendarPath(calendarID)+"/events", accessToken, q, nil, &body); err != nil {
			return Changes{}, err
		}
		for _, w := range body.Items {
			out.Events = append(out.Events, fromWire(w))
		}
		if body.NextPageToken == "" {
			out.NextSyncToken = body.NextSyncToken
			return out, nil
		}
		page = body.NextPageToken
	}
}
