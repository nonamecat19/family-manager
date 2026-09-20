package tasks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/nnc/family-manager/services/telegram/db"
	"github.com/nnc/family-manager/services/telegram/internal/bot"
	"github.com/nnc/family-manager/services/telegram/internal/i18n"
	"github.com/nnc/family-manager/services/telegram/internal/session"
	"github.com/nnc/family-manager/services/telegram/internal/telegram"
)

type fixture struct {
	bot      *bot.Bot
	api      *fakeAPI
	sessions *fakeSessions
	states   *fakeStates
	ran      []string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	api := newFakeAPI(t)
	sessions := newFakeSessions()
	states := &fakeStates{}
	f := &fixture{api: api, sessions: sessions, states: states}

	opts := Bot(&http.Client{}, "http://tasks:8080")
	f.bot = bot.New(bot.Options{
		Name:      "tasks",
		Intro:     opts.Intro,
		API:       api.client(t),
		Sessions:  sessions,
		States:    states,
		Commands:  opts.Commands,
		Callbacks: opts.Callbacks,
		OnText:    opts.OnText,
		Home:      opts.Home,
	})
	return f
}

func TestTasksBotCommandsRegistered(t *testing.T) {
	f := newFixture(t)

	if err := f.bot.Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	f.api.mu.Lock()
	defer f.api.mu.Unlock()
	var names []string
	for _, c := range f.api.commands {
		names = append(names, c.Command)
	}
	joined := strings.Join(names, ",")
	for _, want := range []string{"add", "today", "mine", "done", "help", "menu", "unlink"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("command menu %q is missing %q", joined, want)
		}
	}
	if strings.Contains(joined, "start") {
		t.Fatalf("hidden command leaked into the menu: %q", joined)
	}
}

func TestHelpShowsTasksCommands(t *testing.T) {
	f := newFixture(t)
	f.sessions.linked[42] = &session.Session{UserID: "user-1", AccessToken: "access-1"}

	f.bot.Handle(context.Background(), message("/help"))

	reply := f.api.last(t)
	for _, want := range []string{"/add", "/today", "/mine", "/done", "/menu", "/unlink"} {
		if !strings.Contains(reply, want) {
			t.Fatalf("help %q missing %q", reply, want)
		}
	}
}

func TestAddCommandWithoutArgsShowsUsage(t *testing.T) {
	f := newFixture(t)
	f.sessions.linked[42] = &session.Session{UserID: "user-1", AccessToken: "access-1"}

	f.bot.Handle(context.Background(), message("/add"))

	contains(t, f.api.last(t), i18n.T(i18n.Default, i18n.TasksAddUsage))
}

func TestFreeTextGoesToOnText(t *testing.T) {
	f := newFixture(t)
	f.sessions.linked[42] = &session.Session{UserID: "user-1", AccessToken: "access-1"}

	f.bot.Handle(context.Background(), message("random text"))

	contains(t, f.api.last(t), i18n.T(i18n.Default, i18n.OnlyButtons))
}

type fakeAPI struct {
	server *httptest.Server

	mu       sync.Mutex
	sent     []sentMessage
	toasts   []string
	commands []telegram.BotCommand
}

type sentMessage struct {
	ChatID      int64                          `json:"chat_id"`
	MessageID   int64                          `json:"message_id"`
	Text        string                         `json:"text"`
	ParseMode   string                         `json:"parse_mode"`
	ReplyMarkup *telegram.InlineKeyboardMarkup `json:"reply_markup"`
	edited      bool
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

type fakeSessions struct {
	linked   map[int64]*session.Session
	redeemed []string
	unlinked []int64
	expired  []int64
	err      error
	approved []string
	denied   []string
	decide   error
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

func (f *fakeSessions) ApproveLogin(_ context.Context, id int64, code string) error {
	if _, ok := f.linked[id]; !ok {
		return session.ErrNotLinked
	}
	if f.decide != nil {
		return f.decide
	}
	f.approved = append(f.approved, code)
	return nil
}

func (f *fakeSessions) DenyLogin(_ context.Context, id int64, code string) error {
	if _, ok := f.linked[id]; !ok {
		return session.ErrNotLinked
	}
	if f.decide != nil {
		return f.decide
	}
	f.denied = append(f.denied, code)
	return nil
}

func message(text string) telegram.Update {
	return telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			MessageID: 1,
			From:      &telegram.User{ID: 42, FirstName: "Ada", Username: "ada"},
			Chat:      telegram.Chat{ID: 42, Type: "private"},
			Text:      text,
		},
	}
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
				Chat:      telegram.Chat{ID: id, Type: "private"},
			},
		},
	}
}

func contains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Fatalf("reply %q does not contain %q", haystack, needle)
	}
}