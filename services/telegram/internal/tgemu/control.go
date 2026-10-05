package tgemu

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
)

type sendRequest struct {
	From             User   `json:"from"`
	Chat             *Chat  `json:"chat,omitempty"`
	Text             string `json:"text"`
	ReplyToMessageID int64  `json:"reply_to_message_id,omitempty"`
}

type pressRequest struct {
	From      User   `json:"from"`
	ChatID    int64  `json:"chat_id,omitempty"`
	MessageID int64  `json:"message_id,omitempty"`
	Data      string `json:"data,omitempty"`
	Button    string `json:"button,omitempty"`
}

type botInfo struct {
	Name        string       `json:"name"`
	ID          int64        `json:"id"`
	Username    string       `json:"username"`
	Commands    []BotCommand `json:"commands"`
	Webhook     string       `json:"webhook,omitempty"`
	Pending     int          `json:"pending_updates"`
	Chats       []int64      `json:"chats"`
	LastSeq     int64        `json:"last_seq"`
	HookError   string       `json:"webhook_error,omitempty"`
	CallsServed int          `json:"calls"`
}

func controlError(w http.ResponseWriter, status int, format string, args ...any) {
	writeJSON(w, status, map[string]any{"error": fmt.Sprintf(format, args...)})
}

func (s *Server) routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /_emu/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	})
	mux.HandleFunc("POST /_emu/reset", s.handleReset)
	mux.HandleFunc("GET /_emu/bots", s.handleBots)
	mux.HandleFunc("GET /_emu/bots/{bot}", s.handleBot)
	mux.HandleFunc("POST /_emu/bots/{bot}/messages", s.handleSend)
	mux.HandleFunc("POST /_emu/bots/{bot}/callbacks", s.handlePress)
	mux.HandleFunc("GET /_emu/bots/{bot}/callbacks/{id}", s.handleAnswer)
	mux.HandleFunc("GET /_emu/bots/{bot}/chats/{chat}/messages", s.handleTranscript)
	mux.HandleFunc("GET /_emu/bots/{bot}/events", s.handleEvents)
	mux.HandleFunc("GET /_emu/bots/{bot}/calls", s.handleCalls)
	mux.HandleFunc("POST /_emu/faults", s.handleFault)
	mux.HandleFunc("DELETE /_emu/faults", func(w http.ResponseWriter, _ *http.Request) {
		s.mu.Lock()
		s.faults = nil
		s.mu.Unlock()
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	})
}

func (s *Server) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, b := range s.bots {
		s.clearWebhookLocked(b)
		b.updates = nil
		b.chats = map[int64]*chat{}
		b.answers = map[string]*Answer{}
		b.events = nil
		b.calls = nil
		b.allowed = nil
		b.pollGen++
	}
	s.faults = nil
	s.notify()
}

func (s *Server) handleReset(w http.ResponseWriter, _ *http.Request) {
	s.Reset()
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) infoLocked(b *bot) botInfo {
	chats := make([]int64, 0, len(b.chats))
	for id := range b.chats {
		chats = append(chats, id)
	}
	slices.Sort(chats)
	var last int64
	if n := len(b.events); n > 0 {
		last = b.events[n-1].Seq
	}
	return botInfo{
		Name: b.name, ID: b.id, Username: b.user().Username,
		Commands: append([]BotCommand{}, b.commands...), Webhook: b.webhook,
		Pending: len(b.updates), Chats: chats, LastSeq: last, HookError: b.lastHookError,
		CallsServed: len(b.calls),
	}
}

func (s *Server) handleBots(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []botInfo{}
	for _, b := range s.bots {
		out = append(out, s.infoLocked(b))
	}
	slices.SortFunc(out, func(a, b botInfo) int { return strings.Compare(a.Name, b.Name) })
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleBot(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	writeJSON(w, http.StatusOK, s.infoLocked(s.botLocked(r.PathValue("bot"))))
}

func commandEntities(text string) []Entity {
	if !strings.HasPrefix(text, "/") {
		return nil
	}
	end := strings.IndexAny(text, " \n\t")
	if end < 0 {
		end = len(text)
	}
	return []Entity{{Type: "bot_command", Offset: 0, Length: utf16Len(text[:end])}}
}

func (s *Server) handleSend(w http.ResponseWriter, r *http.Request) {
	var req sendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		controlError(w, http.StatusBadRequest, "decode body: %v", err)
		return
	}
	if req.From.ID == 0 {
		controlError(w, http.StatusBadRequest, "from.id is required")
		return
	}
	if req.Text == "" {
		controlError(w, http.StatusBadRequest, "text is required")
		return
	}
	if req.From.FirstName == "" {
		req.From.FirstName = "User" + strconv.FormatInt(req.From.ID, 10)
	}
	req.From.IsBot = false

	s.mu.Lock()
	defer s.mu.Unlock()
	b := s.botLocked(r.PathValue("bot"))
	info := Chat{ID: req.From.ID, Type: "private", FirstName: req.From.FirstName, Username: req.From.Username}
	if req.Chat != nil && req.Chat.ID != 0 {
		info = *req.Chat
		if info.Type == "" {
			info.Type = "group"
		}
	}
	c, ok := b.chats[info.ID]
	if !ok {
		c = &chat{info: info, nextID: 1}
		b.chats[info.ID] = c
	}
	from := req.From
	msg := Message{
		MessageID: c.nextID,
		From:      &from,
		Chat:      c.info,
		Date:      s.now().Unix(),
		Text:      req.Text,
		Entities:  commandEntities(req.Text),
	}
	if req.ReplyToMessageID != 0 {
		orig := c.find(req.ReplyToMessageID)
		if orig == nil {
			controlError(w, http.StatusBadRequest, "reply_to_message_id %d not found", req.ReplyToMessageID)
			return
		}
		cp := orig.Message
		cp.ReplyToMessage = nil
		msg.ReplyToMessage = &cp
	}
	c.nextID++
	c.records = append(c.records, &Record{Message: msg})
	snapshot := msg
	ev := s.emitLocked(b, Event{Kind: "user_message", ChatID: c.info.ID, Message: &snapshot})
	u := s.enqueueLocked(b, Update{Message: &snapshot})
	writeJSON(w, http.StatusOK, map[string]any{"update_id": u.UpdateID, "seq": ev.Seq, "message": msg})
}

func findButton(m *Markup, data, text string) (string, bool) {
	if m == nil {
		return "", false
	}
	for _, row := range m.InlineKeyboard {
		for _, btn := range row {
			if btn.CallbackData == "" {
				continue
			}
			if (data != "" && btn.CallbackData == data) || (data == "" && text != "" && btn.Text == text) {
				return btn.CallbackData, true
			}
		}
	}
	return "", false
}

func (s *Server) handlePress(w http.ResponseWriter, r *http.Request) {
	var req pressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		controlError(w, http.StatusBadRequest, "decode body: %v", err)
		return
	}
	if req.From.ID == 0 {
		controlError(w, http.StatusBadRequest, "from.id is required")
		return
	}
	if req.Data == "" && req.Button == "" {
		controlError(w, http.StatusBadRequest, "data or button is required")
		return
	}
	if req.ChatID == 0 {
		req.ChatID = req.From.ID
	}
	if req.From.FirstName == "" {
		req.From.FirstName = "User" + strconv.FormatInt(req.From.ID, 10)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	b := s.botLocked(r.PathValue("bot"))
	c, ok := b.chats[req.ChatID]
	if !ok {
		controlError(w, http.StatusNotFound, "chat %d not found", req.ChatID)
		return
	}
	var rec *Record
	var data string
	if req.MessageID != 0 {
		rec = c.find(req.MessageID)
		if rec == nil {
			controlError(w, http.StatusNotFound, "message %d not found", req.MessageID)
			return
		}
		d, found := findButton(rec.Message.ReplyMarkup, req.Data, req.Button)
		if !found {
			controlError(w, http.StatusConflict, "message %d has no button matching data=%q button=%q",
				req.MessageID, req.Data, req.Button)
			return
		}
		data = d
	} else {
		for i := len(c.records) - 1; i >= 0; i-- {
			cand := c.records[i]
			if cand.Deleted || !cand.Outgoing {
				continue
			}
			if d, found := findButton(cand.Message.ReplyMarkup, req.Data, req.Button); found {
				rec, data = cand, d
				break
			}
		}
		if rec == nil {
			controlError(w, http.StatusConflict, "no bot message in chat %d has a button matching data=%q button=%q",
				req.ChatID, req.Data, req.Button)
			return
		}
	}

	b.nextCB++
	id := fmt.Sprintf("%d%06d", b.id, b.nextCB)
	b.answers[id] = &Answer{}
	snapshot := rec.Message
	from := req.From
	from.IsBot = false
	q := CallbackQuery{
		ID: id, From: from, Message: &snapshot,
		ChatInstance: strconv.FormatInt(c.info.ID^b.id, 10), Data: data,
	}
	ev := s.emitLocked(b, Event{Kind: "user_callback", ChatID: c.info.ID, Message: &snapshot,
		CallbackQueryID: id, Data: data})
	u := s.enqueueLocked(b, Update{CallbackQuery: &q})
	writeJSON(w, http.StatusOK, map[string]any{
		"callback_query_id": id, "update_id": u.UpdateID, "seq": ev.Seq, "data": data,
		"message_id": rec.Message.MessageID,
	})
}

func (s *Server) handleAnswer(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b := s.botLocked(r.PathValue("bot"))
	a, ok := b.answers[r.PathValue("id")]
	if !ok {
		controlError(w, http.StatusNotFound, "callback query not found")
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func (s *Server) handleTranscript(w http.ResponseWriter, r *http.Request) {
	chatID, err := strconv.ParseInt(r.PathValue("chat"), 10, 64)
	if err != nil {
		controlError(w, http.StatusBadRequest, "bad chat id")
		return
	}
	all := r.URL.Query().Get("deleted") == "true"
	s.mu.Lock()
	defer s.mu.Unlock()
	b := s.botLocked(r.PathValue("bot"))
	out := []Record{}
	if c, ok := b.chats[chatID]; ok {
		for _, rec := range c.records {
			if rec.Deleted && !all {
				continue
			}
			out = append(out, *rec)
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	after, _ := strconv.ParseInt(q.Get("after"), 10, 64)
	chatID, _ := strconv.ParseInt(q.Get("chat"), 10, 64)
	kinds := map[string]bool{}
	for _, k := range strings.Split(q.Get("kind"), ",") {
		if k = strings.TrimSpace(k); k != "" {
			kinds[k] = true
		}
	}
	timeout := 0 * time.Second
	if t := q.Get("timeout"); t != "" {
		d, err := time.ParseDuration(t)
		if err != nil {
			controlError(w, http.StatusBadRequest, "bad timeout: %v", err)
			return
		}
		timeout = min(d, 120*time.Second)
	}
	deadline := time.Now().Add(timeout)
	name := r.PathValue("bot")

	s.mu.Lock()
	for {
		b := s.botLocked(name)
		out := []Event{}
		for _, e := range b.events {
			if e.Seq <= after {
				continue
			}
			if chatID != 0 && e.ChatID != 0 && e.ChatID != chatID {
				continue
			}
			if len(kinds) > 0 && !kinds[e.Kind] {
				continue
			}
			out = append(out, e)
		}
		if len(out) > 0 || !time.Now().Before(deadline) {
			s.mu.Unlock()
			writeJSON(w, http.StatusOK, out)
			return
		}
		wait := s.changed
		s.mu.Unlock()
		timer := time.NewTimer(time.Until(deadline))
		select {
		case <-r.Context().Done():
			timer.Stop()
			return
		case <-timer.C:
		case <-wait:
			timer.Stop()
		}
		s.mu.Lock()
	}
}

func (s *Server) handleCalls(w http.ResponseWriter, r *http.Request) {
	method := r.URL.Query().Get("method")
	s.mu.Lock()
	defer s.mu.Unlock()
	b := s.botLocked(r.PathValue("bot"))
	out := []Call{}
	for _, c := range b.calls {
		if method == "" || strings.EqualFold(c.Method, method) {
			out = append(out, c)
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleFault(w http.ResponseWriter, r *http.Request) {
	var f Fault
	if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
		controlError(w, http.StatusBadRequest, "decode body: %v", err)
		return
	}
	if f.ErrorCode == 0 {
		f.ErrorCode = http.StatusTooManyRequests
	}
	if f.Description == "" {
		if f.ErrorCode == http.StatusTooManyRequests {
			if f.RetryAfter == 0 {
				f.RetryAfter = 1
			}
			f.Description = fmt.Sprintf("Too Many Requests: retry after %d", f.RetryAfter)
		} else {
			f.Description = http.StatusText(f.ErrorCode)
		}
	}
	if f.Count <= 0 {
		f.Count = 1
	}
	s.mu.Lock()
	s.faults = append(s.faults, &f)
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, f)
}
