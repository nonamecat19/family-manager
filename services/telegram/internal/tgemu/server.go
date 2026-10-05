package tgemu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	maxEvents = 5000
	maxCalls  = 1000
)

var (
	tokenPattern   = regexp.MustCompile(`^(\d+):([A-Za-z0-9_-]+)$`)
	commandPattern = regexp.MustCompile(`^[a-z0-9_]{1,32}$`)
	secretPattern  = regexp.MustCompile(`^[A-Za-z0-9_-]{1,256}$`)
)

type apiError struct {
	code        int
	description string
	retryAfter  int
}

func (e *apiError) Error() string { return e.description }

func badRequest(format string, args ...any) *apiError {
	return &apiError{code: http.StatusBadRequest, description: "Bad Request: " + fmt.Sprintf(format, args...)}
}

type chat struct {
	info    Chat
	nextID  int64
	records []*Record
}

func (c *chat) find(id int64) *Record {
	for _, r := range c.records {
		if r.Message.MessageID == id && !r.Deleted {
			return r
		}
	}
	return nil
}

type bot struct {
	name     string
	id       int64
	commands []BotCommand

	webhook        string
	secret         string
	allowed        []string
	hookCancel     context.CancelFunc
	lastHookError  string
	lastHookErrorT int64

	updates    []Update
	nextUpdate int64
	pollGen    int64

	chats   map[int64]*chat
	answers map[string]*Answer
	nextCB  int64

	events []Event
	calls  []Call
}

func (b *bot) user() User {
	return User{ID: b.id, IsBot: true, FirstName: strings.ToUpper(b.name[:1]) + b.name[1:], Username: b.name + "_bot"}
}

type Server struct {
	mu      sync.Mutex
	bots    map[string]*bot
	seq     int64
	changed chan struct{}
	faults  []*Fault
	client  *http.Client
	log     *slog.Logger
	now     func() time.Time
}

func New(log *slog.Logger) *Server {
	if log == nil {
		log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return &Server{
		bots:    map[string]*bot{},
		changed: make(chan struct{}),
		client:  &http.Client{Timeout: 10 * time.Second},
		log:     log,
		now:     time.Now,
	}
}

func (s *Server) notify() {
	close(s.changed)
	s.changed = make(chan struct{})
}

func (s *Server) botLocked(name string) *bot {
	b, ok := s.bots[name]
	if !ok {
		h := fnv.New32a()
		_, _ = h.Write([]byte(name))
		b = &bot{
			name:       name,
			id:         int64(h.Sum32()%900000000) + 100000000,
			chats:      map[int64]*chat{},
			answers:    map[string]*Answer{},
			nextUpdate: s.now().UnixMilli(),
		}
		s.bots[name] = b
	}
	return b
}

func (s *Server) emitLocked(b *bot, e Event) Event {
	s.seq++
	e.Seq = s.seq
	b.events = append(b.events, e)
	if len(b.events) > maxEvents {
		b.events = b.events[len(b.events)-maxEvents:]
	}
	s.notify()
	return e
}

func (s *Server) enqueueLocked(b *bot, u Update) Update {
	u.UpdateID = b.nextUpdate
	b.nextUpdate++
	b.updates = append(b.updates, u)
	s.notify()
	return u
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	s.routes(mux)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/bot") {
			s.serveBotAPI(w, r)
			return
		}
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(uiHTML)
			return
		}
		http.NotFound(w, r)
	})
	return mux
}

type params map[string]json.RawMessage

func readParams(r *http.Request) (params, error) {
	p := params{}
	addValues := func(v url.Values) {
		for k, vs := range v {
			if len(vs) == 0 {
				continue
			}
			val := vs[len(vs)-1]
			if json.Valid([]byte(val)) && !strings.HasPrefix(val, "\"") {
				p[k] = json.RawMessage(val)
			} else {
				raw, _ := json.Marshal(val)
				p[k] = raw
			}
		}
	}
	addValues(r.URL.Query())
	if r.Method != http.MethodPost {
		return p, nil
	}
	ct := r.Header.Get("Content-Type")
	switch {
	case strings.HasPrefix(ct, "application/json"):
		body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20))
		if err != nil {
			return nil, err
		}
		if len(bytes.TrimSpace(body)) == 0 {
			return p, nil
		}
		var m map[string]json.RawMessage
		if err := json.Unmarshal(body, &m); err != nil {
			return nil, fmt.Errorf("invalid JSON body: %w", err)
		}
		for k, v := range m {
			p[k] = v
		}
	case strings.HasPrefix(ct, "application/x-www-form-urlencoded"), strings.HasPrefix(ct, "multipart/form-data"):
		if err := r.ParseMultipartForm(10 << 20); err != nil && err != http.ErrNotMultipart {
			return nil, err
		}
		addValues(r.PostForm)
	}
	return p, nil
}

func (p params) has(k string) bool {
	v, ok := p[k]
	return ok && string(v) != "null"
}

func (p params) str(k string) string {
	v, ok := p[k]
	if !ok {
		return ""
	}
	var s string
	if json.Unmarshal(v, &s) == nil {
		return s
	}
	return strings.Trim(string(v), "\"")
}

func (p params) int(k string) (int64, error) {
	v, ok := p[k]
	if !ok || string(v) == "null" {
		return 0, nil
	}
	var n int64
	if json.Unmarshal(v, &n) == nil {
		return n, nil
	}
	var str string
	if json.Unmarshal(v, &str) == nil {
		if n, err := strconv.ParseInt(str, 10, 64); err == nil {
			return n, nil
		}
	}
	return 0, badRequest("invalid %s specified", k)
}

func (p params) bool(k string) bool {
	v, ok := p[k]
	if !ok {
		return false
	}
	var b bool
	if json.Unmarshal(v, &b) == nil {
		return b
	}
	return p.str(k) == "true"
}

func (p params) decode(k string, out any) error {
	v, ok := p[k]
	if !ok || string(v) == "null" {
		return nil
	}
	var str string
	if json.Unmarshal(v, &str) == nil {
		v = json.RawMessage(str)
	}
	if err := json.Unmarshal(v, out); err != nil {
		return badRequest("can't parse %s JSON object", k)
	}
	return nil
}

func (p params) plain() map[string]any {
	out := make(map[string]any, len(p))
	for k, v := range p {
		var x any
		if json.Unmarshal(v, &x) == nil {
			out[k] = x
		}
	}
	return out
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeAPIError(w http.ResponseWriter, e *apiError) {
	body := map[string]any{"ok": false, "error_code": e.code, "description": e.description}
	if e.retryAfter > 0 {
		body["parameters"] = map[string]any{"retry_after": e.retryAfter}
	}
	writeJSON(w, e.code, body)
}

func (s *Server) serveBotAPI(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/bot")
	token, method, ok := strings.Cut(rest, "/")
	m := tokenPattern.FindStringSubmatch(token)
	if !ok || m == nil {
		writeAPIError(w, &apiError{code: http.StatusUnauthorized, description: "Unauthorized"})
		return
	}
	id, _ := strconv.ParseInt(m[1], 10, 64)
	name := strings.ToLower(m[2])

	p, err := readParams(r)
	if err != nil {
		writeAPIError(w, badRequest("%s", err.Error()))
		return
	}

	s.mu.Lock()
	b := s.botLocked(name)
	if b.id != id {
		b.id = id
	}
	s.mu.Unlock()

	result, apiErr := s.dispatch(r.Context(), b, strings.ToLower(method), p)

	s.mu.Lock()
	call := Call{At: s.now().UnixMilli(), Method: method, Params: p.plain(), Status: http.StatusOK}
	if apiErr != nil {
		call.Status = apiErr.code
		call.Error = apiErr.description
	}
	b.calls = append(b.calls, call)
	if len(b.calls) > maxCalls {
		b.calls = b.calls[len(b.calls)-maxCalls:]
	}
	s.mu.Unlock()

	if apiErr != nil {
		s.log.Info("bot api error", "bot", name, "method", method, "code", apiErr.code, "error", apiErr.description)
		writeAPIError(w, apiErr)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "result": result})
}

func (s *Server) takeFaultLocked(name, method string) *apiError {
	for i, f := range s.faults {
		if (f.Bot == "" || f.Bot == name) && (f.Method == "" || strings.EqualFold(f.Method, method)) {
			f.Count--
			if f.Count <= 0 {
				s.faults = slices.Delete(s.faults, i, i+1)
			}
			return &apiError{code: f.ErrorCode, description: f.Description, retryAfter: f.RetryAfter}
		}
	}
	return nil
}

func (s *Server) dispatch(ctx context.Context, b *bot, method string, p params) (any, *apiError) {
	s.mu.Lock()
	fault := s.takeFaultLocked(b.name, method)
	s.mu.Unlock()
	if fault != nil {
		return nil, fault
	}

	switch method {
	case "getme":
		s.mu.Lock()
		defer s.mu.Unlock()
		u := b.user()
		return map[string]any{
			"id": u.ID, "is_bot": true, "first_name": u.FirstName, "username": u.Username,
			"can_join_groups": true, "can_read_all_group_messages": false, "supports_inline_queries": false,
		}, nil
	case "getupdates":
		return s.getUpdates(ctx, b, p)
	case "sendmessage":
		return s.sendMessage(b, p)
	case "editmessagetext":
		return s.editMessage(b, p, true)
	case "editmessagereplymarkup":
		return s.editMessage(b, p, false)
	case "deletemessage":
		return s.deleteMessage(b, p)
	case "answercallbackquery":
		return s.answerCallback(b, p)
	case "setmycommands":
		return s.setCommands(b, p)
	case "getmycommands":
		s.mu.Lock()
		defer s.mu.Unlock()
		return append([]BotCommand{}, b.commands...), nil
	case "deletemycommands":
		s.mu.Lock()
		defer s.mu.Unlock()
		b.commands = nil
		return true, nil
	case "setwebhook":
		return s.setWebhook(b, p)
	case "deletewebhook":
		s.mu.Lock()
		defer s.mu.Unlock()
		s.clearWebhookLocked(b)
		if p.bool("drop_pending_updates") {
			b.updates = nil
		}
		return true, nil
	case "getwebhookinfo":
		s.mu.Lock()
		defer s.mu.Unlock()
		info := map[string]any{"url": b.webhook, "has_custom_certificate": false, "pending_update_count": len(b.updates)}
		if b.lastHookError != "" {
			info["last_error_message"] = b.lastHookError
			info["last_error_date"] = b.lastHookErrorT
		}
		if len(b.allowed) > 0 {
			info["allowed_updates"] = b.allowed
		}
		return info, nil
	case "sendchataction":
		return true, nil
	case "close", "logout":
		return true, nil
	}
	return nil, &apiError{code: http.StatusNotFound, description: "Not Found"}
}

func allowedUpdate(allowed []string, u Update) bool {
	if len(allowed) == 0 {
		return true
	}
	return slices.Contains(allowed, u.kind())
}

func (s *Server) getUpdates(ctx context.Context, b *bot, p params) (any, *apiError) {
	offset, err := p.int("offset")
	if err != nil {
		return nil, err.(*apiError)
	}
	limit, err := p.int("limit")
	if err != nil {
		return nil, err.(*apiError)
	}
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	timeout, err := p.int("timeout")
	if err != nil {
		return nil, err.(*apiError)
	}
	var allowed []string
	if derr := p.decode("allowed_updates", &allowed); derr != nil {
		return nil, derr.(*apiError)
	}

	deadline := time.Now().Add(time.Duration(timeout) * time.Second)

	s.mu.Lock()
	if b.webhook != "" {
		s.mu.Unlock()
		return nil, &apiError{code: http.StatusConflict,
			description: "Conflict: can't use getUpdates method while webhook is active; use deleteWebhook to delete the webhook first"}
	}
	if p.has("allowed_updates") {
		b.allowed = allowed
	}
	b.pollGen++
	gen := b.pollGen
	s.notify()
	for {
		if b.pollGen != gen {
			s.mu.Unlock()
			return nil, &apiError{code: http.StatusConflict,
				description: "Conflict: terminated by other getUpdates request; make sure that only one bot instance is running"}
		}
		if offset > 0 {
			kept := b.updates[:0]
			for _, u := range b.updates {
				if u.UpdateID >= offset {
					kept = append(kept, u)
				}
			}
			b.updates = kept
		}
		out := []Update{}
		for _, u := range b.updates {
			if allowedUpdate(b.allowed, u) {
				out = append(out, u)
			}
			if int64(len(out)) >= limit {
				break
			}
		}
		if len(out) > 0 || !time.Now().Before(deadline) {
			s.mu.Unlock()
			return out, nil
		}
		wait := s.changed
		s.mu.Unlock()
		timer := time.NewTimer(time.Until(deadline))
		select {
		case <-ctx.Done():
			timer.Stop()
			return []Update{}, nil
		case <-timer.C:
		case <-wait:
			timer.Stop()
		}
		s.mu.Lock()
	}
}

func (s *Server) chatParam(b *bot, p params) (*chat, *apiError) {
	if !p.has("chat_id") {
		return nil, badRequest("chat_id is empty")
	}
	id, err := p.int("chat_id")
	if err != nil {
		return nil, badRequest("chat not found")
	}
	c, ok := b.chats[id]
	if !ok {
		return nil, badRequest("chat not found")
	}
	return c, nil
}

func (s *Server) sendMessage(b *bot, p params) (any, *apiError) {
	text := p.str("text")
	mode := p.str("parse_mode")
	var markup *Markup
	if err := p.decode("reply_markup", &markup); err != nil {
		return nil, err.(*apiError)
	}
	if err := checkText(text, mode); err != nil {
		return nil, badRequest("%s", err.Error())
	}
	if err := checkMarkup(markup); err != nil {
		return nil, badRequest("%s", err.Error())
	}
	replyTo, err := p.int("reply_to_message_id")
	if err != nil {
		return nil, err.(*apiError)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	c, apiErr := s.chatParam(b, p)
	if apiErr != nil {
		return nil, apiErr
	}
	from := b.user()
	msg := Message{
		MessageID:   c.nextID,
		From:        &from,
		Chat:        c.info,
		Date:        s.now().Unix(),
		Text:        text,
		ReplyMarkup: markup,
	}
	if replyTo != 0 {
		orig := c.find(replyTo)
		if orig == nil {
			return nil, badRequest("message to be replied not found")
		}
		cp := orig.Message
		cp.ReplyToMessage = nil
		msg.ReplyToMessage = &cp
	}
	c.nextID++
	c.records = append(c.records, &Record{Message: msg, ParseMode: mode, Outgoing: true})
	snapshot := msg
	s.emitLocked(b, Event{Kind: "bot_message", ChatID: c.info.ID, Message: &snapshot, ParseMode: mode})
	return msg, nil
}

func sameMarkup(a, b *Markup) bool {
	ja, _ := json.Marshal(a)
	jb, _ := json.Marshal(b)
	if a == nil || len(a.InlineKeyboard) == 0 {
		ja = []byte("null")
	}
	if b == nil || len(b.InlineKeyboard) == 0 {
		jb = []byte("null")
	}
	return bytes.Equal(ja, jb)
}

func (s *Server) editMessage(b *bot, p params, withText bool) (any, *apiError) {
	if p.has("inline_message_id") {
		return nil, badRequest("inline messages are not supported by the emulator")
	}
	var markup *Markup
	if err := p.decode("reply_markup", &markup); err != nil {
		return nil, err.(*apiError)
	}
	if err := checkMarkup(markup); err != nil {
		return nil, badRequest("%s", err.Error())
	}
	text := p.str("text")
	mode := p.str("parse_mode")
	if withText {
		if err := checkText(text, mode); err != nil {
			return nil, badRequest("%s", err.Error())
		}
	}
	id, err := p.int("message_id")
	if err != nil {
		return nil, err.(*apiError)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	c, apiErr := s.chatParam(b, p)
	if apiErr != nil {
		return nil, apiErr
	}
	rec := c.find(id)
	if rec == nil {
		return nil, badRequest("message to edit not found")
	}
	if !rec.Outgoing {
		return nil, badRequest("message can't be edited")
	}
	newText := rec.Message.Text
	newMode := rec.ParseMode
	if withText {
		newText, newMode = text, mode
	}
	if newText == rec.Message.Text && newMode == rec.ParseMode && sameMarkup(markup, rec.Message.ReplyMarkup) {
		return nil, badRequest("message is not modified: specified new message content and reply markup " +
			"are exactly the same as a current content and reply markup of the message")
	}
	if newText != rec.Message.Text {
		rec.History = append(rec.History, rec.Message.Text)
	}
	rec.Message.Text = newText
	rec.ParseMode = newMode
	rec.Message.ReplyMarkup = markup
	rec.Message.EditDate = s.now().Unix()
	snapshot := rec.Message
	s.emitLocked(b, Event{Kind: "bot_edit", ChatID: c.info.ID, Message: &snapshot, ParseMode: newMode})
	return rec.Message, nil
}

func (s *Server) deleteMessage(b *bot, p params) (any, *apiError) {
	id, err := p.int("message_id")
	if err != nil {
		return nil, err.(*apiError)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	c, apiErr := s.chatParam(b, p)
	if apiErr != nil {
		return nil, apiErr
	}
	rec := c.find(id)
	if rec == nil {
		return nil, badRequest("message to delete not found")
	}
	rec.Deleted = true
	snapshot := rec.Message
	s.emitLocked(b, Event{Kind: "bot_delete", ChatID: c.info.ID, Message: &snapshot})
	return true, nil
}

func (s *Server) answerCallback(b *bot, p params) (any, *apiError) {
	id := p.str("callback_query_id")
	text := p.str("text")
	if utf16Len(text) > 200 {
		return nil, badRequest("MESSAGE_TOO_LONG")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := b.answers[id]
	if !ok || a.Answered {
		return nil, badRequest("query is too old and response timeout expired or query ID is invalid")
	}
	a.Answered = true
	a.Text = text
	a.ShowAlert = p.bool("show_alert")
	a.URL = p.str("url")
	s.emitLocked(b, Event{Kind: "callback_answer", CallbackQueryID: id, Text: text, ShowAlert: a.ShowAlert})
	return true, nil
}

func (s *Server) setCommands(b *bot, p params) (any, *apiError) {
	var cmds []BotCommand
	if err := p.decode("commands", &cmds); err != nil {
		return nil, err.(*apiError)
	}
	if len(cmds) > 100 {
		return nil, badRequest("too many commands specified")
	}
	for _, c := range cmds {
		if !commandPattern.MatchString(c.Command) {
			return nil, badRequest("BOT_COMMAND_INVALID")
		}
		if n := utf16Len(c.Description); n < 1 || n > 256 {
			return nil, badRequest("BOT_COMMAND_DESCRIPTION_INVALID")
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	b.commands = cmds
	return true, nil
}

func (s *Server) clearWebhookLocked(b *bot) {
	if b.hookCancel != nil {
		b.hookCancel()
		b.hookCancel = nil
	}
	b.webhook, b.secret = "", ""
}

func (s *Server) setWebhook(b *bot, p params) (any, *apiError) {
	hook := p.str("url")
	secret := p.str("secret_token")
	var allowed []string
	if err := p.decode("allowed_updates", &allowed); err != nil {
		return nil, err.(*apiError)
	}
	if secret != "" && !secretPattern.MatchString(secret) {
		return nil, badRequest("secret token contains unallowed characters")
	}
	if hook != "" {
		u, err := url.Parse(hook)
		if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
			return nil, badRequest("bad webhook: Failed to resolve host: Name or service not known")
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clearWebhookLocked(b)
	if p.bool("drop_pending_updates") {
		b.updates = nil
	}
	if hook == "" {
		return true, nil
	}
	b.webhook, b.secret = hook, secret
	if p.has("allowed_updates") {
		b.allowed = allowed
	}
	b.pollGen++
	ctx, cancel := context.WithCancel(context.Background())
	b.hookCancel = cancel
	go s.deliver(ctx, b)
	s.notify()
	return true, nil
}

func (s *Server) deliver(ctx context.Context, b *bot) {
	backoff := 200 * time.Millisecond
	for {
		s.mu.Lock()
		var next *Update
		for len(b.updates) > 0 {
			if allowedUpdate(b.allowed, b.updates[0]) {
				u := b.updates[0]
				next = &u
				break
			}
			b.updates = b.updates[1:]
		}
		hook, secret := b.webhook, b.secret
		wait := s.changed
		s.mu.Unlock()

		if ctx.Err() != nil {
			return
		}
		if next == nil {
			select {
			case <-ctx.Done():
				return
			case <-wait:
			}
			continue
		}

		err := s.post(ctx, hook, secret, *next)
		s.mu.Lock()
		if err == nil {
			if len(b.updates) > 0 && b.updates[0].UpdateID == next.UpdateID {
				b.updates = b.updates[1:]
			}
			b.lastHookError = ""
			backoff = 200 * time.Millisecond
		} else {
			b.lastHookError = err.Error()
			b.lastHookErrorT = s.now().Unix()
		}
		s.mu.Unlock()
		if err != nil {
			s.log.Warn("webhook delivery failed", "bot", b.name, "error", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			backoff = min(backoff*2, 5*time.Second)
		}
	}
}

func (s *Server) post(ctx context.Context, hook, secret string, u Update) error {
	body, _ := json.Marshal(u)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, hook, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if secret != "" {
		req.Header.Set("X-Telegram-Bot-Api-Secret-Token", secret)
	}
	res, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, res.Body)
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return fmt.Errorf("Wrong response from the webhook: %s", res.Status)
	}
	return nil
}
