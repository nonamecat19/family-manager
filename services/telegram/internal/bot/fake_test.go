package bot

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/nnc/family-manager/services/telegram/db"

	"github.com/nnc/family-manager/services/telegram/internal/session"
	"github.com/nnc/family-manager/services/telegram/internal/telegram"
)

type sentMessage struct {
	ChatID      int64                          `json:"chat_id"`
	MessageID   int64                          `json:"message_id"`
	Text        string                         `json:"text"`
	ParseMode   string                         `json:"parse_mode"`
	ReplyMarkup *telegram.InlineKeyboardMarkup `json:"reply_markup"`
	edited      bool
}

type fakeAPI struct {
	server *httptest.Server

	mu       sync.Mutex
	sent     []sentMessage
	toasts   []string
	commands []telegram.BotCommand
}

func newFakeAPI(t *testing.T) *fakeAPI {
	t.Helper()
	f := &fakeAPI{}

	mux := http.NewServeMux()
	mux.HandleFunc("/bottest-token/getMe", func(w http.ResponseWriter, _ *http.Request) {
		writeResult(w, telegram.User{ID: 1, IsBot: true, Username: "fm_test_bot"})
	})
	mux.HandleFunc("/bottest-token/sendMessage", func(w http.ResponseWriter, r *http.Request) {
		var msg sentMessage
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		f.mu.Lock()
		f.sent = append(f.sent, msg)
		f.mu.Unlock()
		writeResult(w, telegram.Message{MessageID: int64(len(f.sent))})
	})
	mux.HandleFunc("/bottest-token/editMessageText", func(w http.ResponseWriter, r *http.Request) {
		var msg sentMessage
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		msg.edited = true
		f.mu.Lock()
		f.sent = append(f.sent, msg)
		f.mu.Unlock()
		writeResult(w, telegram.Message{MessageID: msg.MessageID})
	})
	mux.HandleFunc("/bottest-token/setMyCommands", func(w http.ResponseWriter, r *http.Request) {
		var params struct {
			Commands []telegram.BotCommand `json:"commands"`
		}
		_ = json.NewDecoder(r.Body).Decode(&params)
		f.mu.Lock()
		f.commands = params.Commands
		f.mu.Unlock()
		writeResult(w, true)
	})
	mux.HandleFunc("/bottest-token/answerCallbackQuery", func(w http.ResponseWriter, r *http.Request) {
		var params struct {
			Text string `json:"text"`
		}
		_ = json.NewDecoder(r.Body).Decode(&params)
		f.mu.Lock()
		f.toasts = append(f.toasts, params.Text)
		f.mu.Unlock()
		writeResult(w, true)
	})

	f.server = httptest.NewServer(mux)
	t.Cleanup(f.server.Close)
	return f
}

func writeResult(w http.ResponseWriter, result any) {
	body, err := json.Marshal(result)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"ok":true,"result":` + string(body) + `}`))
}

func (f *fakeAPI) client(t *testing.T) *telegram.Client {
	t.Helper()
	client, err := telegram.New(telegram.Options{Token: "test-token", APIBase: f.server.URL})
	if err != nil {
		t.Fatalf("telegram.New: %v", err)
	}
	return client
}

func (f *fakeAPI) texts() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.sent))
	for i, m := range f.sent {
		out[i] = m.Text
	}
	return out
}

func (f *fakeAPI) lastMessage(t *testing.T) sentMessage {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.sent) == 0 {
		t.Fatal("the bot sent nothing")
	}
	return f.sent[len(f.sent)-1]
}

func (f *fakeAPI) buttons(t *testing.T) []string {
	t.Helper()
	msg := f.lastMessage(t)
	if msg.ReplyMarkup == nil {
		return nil
	}
	var out []string
	for _, row := range msg.ReplyMarkup.InlineKeyboard {
		for _, b := range row {
			out = append(out, b.Text+"="+b.CallbackData)
		}
	}
	return out
}

func (f *fakeAPI) lastToast(t *testing.T) string {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.toasts) == 0 {
		t.Fatal("no callback was answered")
	}
	return f.toasts[len(f.toasts)-1]
}

func (f *fakeAPI) last(t *testing.T) string {
	t.Helper()
	texts := f.texts()
	if len(texts) == 0 {
		t.Fatal("the bot sent nothing")
	}
	return texts[len(texts)-1]
}

type fakeStates struct {
	kind    string
	payload map[string]string
	cleared int
	set     int
}

func (f *fakeStates) UpsertChatState(
	_ context.Context, arg db.UpsertChatStateParams,
) (db.ChatState, error) {
	f.set++
	f.kind = arg.Kind
	f.payload = map[string]string{}
	if len(arg.Payload) > 0 {
		_ = json.Unmarshal(arg.Payload, &f.payload)
	}
	return db.ChatState{Bot: arg.Bot, Kind: arg.Kind, Payload: arg.Payload}, nil
}

func (f *fakeStates) GetChatState(
	_ context.Context, _ db.GetChatStateParams,
) (db.ChatState, error) {
	if f.kind == "" {
		return db.ChatState{}, pgx.ErrNoRows
	}
	body, _ := json.Marshal(f.payload)
	return db.ChatState{Kind: f.kind, Payload: body}, nil
}

func (f *fakeStates) ClearChatState(_ context.Context, _ db.ClearChatStateParams) (int64, error) {
	f.cleared++
	f.kind = ""
	f.payload = nil
	return 1, nil
}

func callback(data string, userID ...int64) telegram.Update {
	id := int64(42)
	if len(userID) > 0 {
		id = userID[0]
	}
	return telegram.Update{
		UpdateID: 2,
		CallbackQuery: &telegram.CallbackQuery{
			ID:   "cbq-1",
			From: telegram.User{ID: id, FirstName: "Ada", Username: "ada"},
			Data: data,
			Message: &telegram.Message{
				MessageID: 555,
				Chat:      telegram.Chat{ID: 99, Type: "private"},
			},
		},
	}
}

type fakeSessions struct {
	linked   map[int64]*session.Session
	redeemed []string
	unlinked []int64
	expired  []int64
	err      error
}

func newFakeSessions() *fakeSessions {
	return &fakeSessions{linked: map[int64]*session.Session{}}
}

func (f *fakeSessions) Session(_ context.Context, id int64) (*session.Session, error) {
	if f.err != nil {
		return nil, f.err
	}
	s, ok := f.linked[id]
	if !ok {
		return nil, session.ErrNotLinked
	}
	return s, nil
}

func (f *fakeSessions) Redeem(
	_ context.Context, linkToken string, from telegram.User, _ int64,
) (*session.Session, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.redeemed = append(f.redeemed, linkToken)
	s := &session.Session{UserID: "user-1", AccessToken: "access-1"}
	f.linked[from.ID] = s
	return s, nil
}

func (f *fakeSessions) ExpireAccess(_ context.Context, id int64) error {
	f.expired = append(f.expired, id)
	return nil
}

func (f *fakeSessions) Unlink(_ context.Context, id int64) error {
	if _, ok := f.linked[id]; !ok {
		return session.ErrNotLinked
	}
	delete(f.linked, id)
	f.unlinked = append(f.unlinked, id)
	return nil
}

func message(text string) telegram.Update {
	return telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 1,
			From:      &telegram.User{ID: 42, FirstName: "Ada", Username: "ada"},
			Chat:      telegram.Chat{ID: 99, Type: "private"},
			Text:      text,
		},
	}
}

func contains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Fatalf("reply %q does not contain %q", haystack, needle)
	}
}

var errBoom = errors.New("boom")

type fakePrefs struct {
	locale string
	saved  []string
	err    error
}

func (f *fakePrefs) Locale(context.Context, string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.locale, nil
}

func (f *fakePrefs) SetLocale(_ context.Context, _, locale string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	f.saved = append(f.saved, locale)
	f.locale = locale
	return locale, nil
}
