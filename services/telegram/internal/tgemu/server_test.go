package tgemu_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nnc/family-manager/services/telegram/internal/telegram"
	"github.com/nnc/family-manager/services/telegram/internal/tgemu"
)

const userID = 4242

type harness struct {
	t   *testing.T
	srv *httptest.Server
	api *telegram.Client
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	srv := httptest.NewServer(tgemu.New(nil).Handler())
	t.Cleanup(srv.Close)
	api, err := telegram.New(telegram.Options{Token: "1001:finance", APIBase: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	return &harness{t: t, srv: srv, api: api}
}

func (h *harness) control(method, path string, body any, out any) int {
	h.t.Helper()
	var r io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		r = bytes.NewReader(raw)
	}
	req, _ := http.NewRequest(method, h.srv.URL+path, r)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		h.t.Fatal(err)
	}
	defer res.Body.Close()
	if out != nil {
		if err := json.NewDecoder(res.Body).Decode(out); err != nil {
			h.t.Fatalf("decode %s: %v", path, err)
		}
	}
	return res.StatusCode
}

func (h *harness) say(text string) {
	h.t.Helper()
	if code := h.control("POST", "/_emu/bots/finance/messages",
		map[string]any{"from": map[string]any{"id": userID, "first_name": "Ann"}, "text": text}, nil); code != 200 {
		h.t.Fatalf("say: status %d", code)
	}
}

func ctx(t *testing.T) context.Context {
	c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return c
}

func TestGetMeDerivesIdentityFromToken(t *testing.T) {
	h := newHarness(t)
	me, err := h.api.GetMe(ctx(t))
	if err != nil {
		t.Fatal(err)
	}
	if me.ID != 1001 || me.Username != "finance_bot" || !me.IsBot {
		t.Fatalf("unexpected me: %+v", me)
	}
}

func TestBadTokenIsUnauthorized(t *testing.T) {
	h := newHarness(t)
	res, err := http.Get(h.srv.URL + "/botnot-a-token/getMe")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status %d", res.StatusCode)
	}
}

func TestUserMessageReachesLongPoll(t *testing.T) {
	h := newHarness(t)
	done := make(chan []telegram.Update, 1)
	go func() {
		ups, err := h.api.GetUpdates(ctx(t), 0, 3*time.Second)
		if err != nil {
			t.Error(err)
		}
		done <- ups
	}()
	time.Sleep(100 * time.Millisecond)
	h.say("/start abc")

	ups := <-done
	if len(ups) != 1 || ups[0].Message == nil {
		t.Fatalf("updates: %+v", ups)
	}
	m := ups[0].Message
	if m.Text != "/start abc" || m.Chat.ID != userID || m.From.ID != userID {
		t.Fatalf("message: %+v", m)
	}
	if len(m.Entities) != 1 || m.Entities[0].Type != "bot_command" || m.Entities[0].Length != 6 {
		t.Fatalf("entities: %+v", m.Entities)
	}

	again, err := h.api.GetUpdates(ctx(t), ups[0].UpdateID+1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(again) != 0 {
		t.Fatalf("offset did not acknowledge: %+v", again)
	}
}

func TestSendRequiresKnownChat(t *testing.T) {
	h := newHarness(t)
	_, err := h.api.SendMessage(ctx(t), telegram.SendMessageParams{ChatID: 999, Text: "hi"})
	if err == nil || !strings.Contains(err.Error(), "chat not found") {
		t.Fatalf("err = %v", err)
	}
}

func TestHTMLValidation(t *testing.T) {
	h := newHarness(t)
	h.say("hi")
	cases := map[string]string{
		"<b>open":                 "Can't find end tag",
		"<b>x</i>":                "Unmatched end tag",
		"<div>x</div>":            "Unsupported start tag",
		"a < b":                   "Unclosed start tag",
		"<b></b>":                 "message text is empty",
		strings.Repeat("x", 4097): "message is too long",
	}
	for text, want := range cases {
		_, err := h.api.SendMessage(ctx(t), telegram.SendMessageParams{ChatID: userID, Text: text, ParseMode: "HTML"})
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: err = %v, want %q", text[:min(len(text), 20)], err, want)
		}
	}
	ok := []string{"<b>bold</b> &amp; <i>it</i>", "1 &lt; 2", `<a href="https://x.y">x</a>`, "Tom & Jerry", "<pre><code>x</code></pre>"}
	for _, text := range ok {
		if _, err := h.api.SendMessage(ctx(t), telegram.SendMessageParams{ChatID: userID, Text: text, ParseMode: "HTML"}); err != nil {
			t.Errorf("%q rejected: %v", text, err)
		}
	}
}

func TestKeyboardValidation(t *testing.T) {
	h := newHarness(t)
	h.say("hi")
	long := strings.Repeat("d", 65)
	_, err := h.api.SendMessage(ctx(t), telegram.SendMessageParams{ChatID: userID, Text: "x",
		ReplyMarkup: &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{{{Text: "a", CallbackData: long}}}}})
	if err == nil || !strings.Contains(err.Error(), "BUTTON_DATA_INVALID") {
		t.Fatalf("err = %v", err)
	}
	_, err = h.api.SendMessage(ctx(t), telegram.SendMessageParams{ChatID: userID, Text: "x",
		ReplyMarkup: &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{{{Text: "", CallbackData: "noop"}}}}})
	if err == nil || !strings.Contains(err.Error(), "BUTTON_TEXT_EMPTY") {
		t.Fatalf("empty text: err = %v", err)
	}
	_, err = h.api.SendMessage(ctx(t), telegram.SendMessageParams{ChatID: userID, Text: "x",
		ReplyMarkup: &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{{{Text: " ", CallbackData: "noop"}}}}})
	if err != nil {
		t.Fatalf("spacer button rejected: %v", err)
	}
}

func TestEditNotModified(t *testing.T) {
	h := newHarness(t)
	h.say("hi")
	sent, err := h.api.SendMessage(ctx(t), telegram.SendMessageParams{ChatID: userID, Text: "one"})
	if err != nil {
		t.Fatal(err)
	}
	err = h.api.EditMessageText(ctx(t), telegram.EditMessageTextParams{ChatID: userID, MessageID: sent.MessageID, Text: "two"})
	if err != nil {
		t.Fatal(err)
	}
	var recs []tgemu.Record
	h.control("GET", "/_emu/bots/finance/chats/4242/messages", nil, &recs)
	if len(recs) != 2 || recs[1].Message.Text != "two" || len(recs[1].History) != 1 {
		t.Fatalf("transcript: %+v", recs)
	}

	res, err := http.Post(h.srv.URL+"/bot1001:finance/editMessageText", "application/json",
		strings.NewReader(`{"chat_id":4242,"message_id":2,"text":"two"}`))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 400 || !strings.Contains(string(body), "message is not modified") {
		t.Fatalf("status %d body %s", res.StatusCode, body)
	}

	err = h.api.EditMessageText(ctx(t), telegram.EditMessageTextParams{ChatID: userID, MessageID: 1, Text: "x"})
	if err == nil || !strings.Contains(err.Error(), "can't be edited") {
		t.Fatalf("editing user message: %v", err)
	}
}

func TestCallbackRoundTrip(t *testing.T) {
	h := newHarness(t)
	h.say("hi")
	sent, err := h.api.SendMessage(ctx(t), telegram.SendMessageParams{ChatID: userID, Text: "pick",
		ReplyMarkup: &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{{Text: "Yes", CallbackData: "c:yes"}, {Text: "No", CallbackData: "c:no"}}}}})
	if err != nil {
		t.Fatal(err)
	}

	var pressed struct {
		ID   string `json:"callback_query_id"`
		Data string `json:"data"`
	}
	code := h.control("POST", "/_emu/bots/finance/callbacks", map[string]any{
		"from": map[string]any{"id": userID}, "button": "No"}, &pressed)
	if code != 200 || pressed.Data != "c:no" {
		t.Fatalf("press: %d %+v", code, pressed)
	}
	if code := h.control("POST", "/_emu/bots/finance/callbacks", map[string]any{
		"from": map[string]any{"id": userID}, "data": "c:maybe"}, nil); code != http.StatusConflict {
		t.Fatalf("pressing a missing button: %d", code)
	}

	ups, err := h.api.GetUpdates(ctx(t), 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	cb := ups[len(ups)-1].CallbackQuery
	if cb == nil || cb.Data != "c:no" || cb.Message.MessageID != sent.MessageID {
		t.Fatalf("callback update: %+v", ups)
	}
	if err := h.api.AnswerCallbackQuery(ctx(t), cb.ID, "Saved", false); err != nil {
		t.Fatal(err)
	}
	if err := h.api.AnswerCallbackQuery(ctx(t), cb.ID, "again", false); err == nil {
		t.Fatal("second answer accepted")
	}
	var ans tgemu.Answer
	h.control("GET", "/_emu/bots/finance/callbacks/"+cb.ID, nil, &ans)
	if !ans.Answered || ans.Text != "Saved" {
		t.Fatalf("answer: %+v", ans)
	}
}

func TestEventsLongPoll(t *testing.T) {
	h := newHarness(t)
	h.say("hi")
	go func() {
		time.Sleep(150 * time.Millisecond)
		_, _ = h.api.SendMessage(context.Background(), telegram.SendMessageParams{ChatID: userID, Text: "reply"})
	}()
	var evs []tgemu.Event
	h.control("GET", "/_emu/bots/finance/events?after=1&kind=bot_message&timeout=3s", nil, &evs)
	if len(evs) != 1 || evs[0].Message.Text != "reply" {
		t.Fatalf("events: %+v", evs)
	}
}

func TestFaultInjectionRetryAfter(t *testing.T) {
	h := newHarness(t)
	h.say("hi")
	h.control("POST", "/_emu/faults", map[string]any{"method": "sendMessage", "retry_after": 3}, nil)
	_, err := h.api.SendMessage(ctx(t), telegram.SendMessageParams{ChatID: userID, Text: "x"})
	if d, ok := telegram.RetryAfter(err); !ok || d != 3*time.Second {
		t.Fatalf("retry after = %v %v (err %v)", d, ok, err)
	}
	if _, err := h.api.SendMessage(ctx(t), telegram.SendMessageParams{ChatID: userID, Text: "x"}); err != nil {
		t.Fatalf("fault not consumed: %v", err)
	}
}

func TestSetMyCommandsValidates(t *testing.T) {
	h := newHarness(t)
	if err := h.api.SetMyCommands(ctx(t), []telegram.BotCommand{{Command: "Bad-Name", Description: "x"}}); err == nil {
		t.Fatal("invalid command accepted")
	}
	if err := h.api.SetMyCommands(ctx(t), []telegram.BotCommand{{Command: "menu", Description: "open menu"}}); err != nil {
		t.Fatal(err)
	}
	var info struct {
		Commands []tgemu.BotCommand `json:"commands"`
	}
	h.control("GET", "/_emu/bots/finance", nil, &info)
	if len(info.Commands) != 1 || info.Commands[0].Command != "menu" {
		t.Fatalf("commands: %+v", info.Commands)
	}
}

func TestWebhookDeliveryAndConflict(t *testing.T) {
	h := newHarness(t)
	var got atomic.Value
	var secret atomic.Value
	hook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		secret.Store(r.Header.Get("X-Telegram-Bot-Api-Secret-Token"))
		var u telegram.Update
		_ = json.NewDecoder(r.Body).Decode(&u)
		got.Store(u)
	}))
	t.Cleanup(hook.Close)

	if err := h.api.SetWebhook(ctx(t), hook.URL+"/hook", "s3cret"); err != nil {
		t.Fatal(err)
	}
	if _, err := h.api.GetUpdates(ctx(t), 0, 0); err == nil || !strings.Contains(err.Error(), "webhook is active") {
		t.Fatalf("getUpdates with webhook: %v", err)
	}
	h.say("via hook")
	deadline := time.Now().Add(3 * time.Second)
	for got.Load() == nil && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	u, _ := got.Load().(telegram.Update)
	if u.Message == nil || u.Message.Text != "via hook" || secret.Load() != "s3cret" {
		t.Fatalf("delivered %+v secret %v", u, secret.Load())
	}
	if err := h.api.DeleteWebhook(ctx(t)); err != nil {
		t.Fatal(err)
	}
	if _, err := h.api.GetUpdates(ctx(t), 0, 0); err != nil {
		t.Fatalf("getUpdates after deleteWebhook: %v", err)
	}
}

func TestConcurrentPollsConflict(t *testing.T) {
	h := newHarness(t)
	first := make(chan error, 1)
	go func() {
		_, err := h.api.GetUpdates(ctx(t), 0, 3*time.Second)
		first <- err
	}()
	time.Sleep(100 * time.Millisecond)
	go func() { _, _ = h.api.GetUpdates(ctx(t), 0, 1*time.Second) }()
	if err := <-first; err == nil || !strings.Contains(err.Error(), "terminated by other getUpdates") {
		t.Fatalf("first poll: %v", err)
	}
}

func TestReset(t *testing.T) {
	h := newHarness(t)
	h.say("hi")
	h.control("POST", "/_emu/reset", nil, nil)
	var recs []tgemu.Record
	h.control("GET", "/_emu/bots/finance/chats/4242/messages", nil, &recs)
	if len(recs) != 0 {
		t.Fatalf("transcript after reset: %+v", recs)
	}
}

func TestUpdateIDsSurviveRestart(t *testing.T) {
	first := newHarness(t)
	first.say("one")
	ups, err := first.api.GetUpdates(ctx(t), 0, 0)
	if err != nil || len(ups) != 1 {
		t.Fatalf("first: %v %+v", err, ups)
	}
	persisted := ups[0].UpdateID + 1

	time.Sleep(5 * time.Millisecond)
	second := newHarness(t)
	second.say("two")
	ups, err = second.api.GetUpdates(ctx(t), persisted, 0)
	if err != nil || len(ups) != 1 || ups[0].Message.Text != "two" {
		t.Fatalf("restarted emulator dropped update below persisted offset %d: %v %+v", persisted, err, ups)
	}
}
