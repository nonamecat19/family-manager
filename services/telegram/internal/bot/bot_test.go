package bot

import (
	"context"
	"strings"
	"testing"

	"github.com/nnc/family-manager/services/telegram/internal/i18n"

	"github.com/nnc/family-manager/services/telegram/internal/session"
	"github.com/nnc/family-manager/services/telegram/internal/telegram"
)

type fixture struct {
	bot      *Bot
	api      *fakeAPI
	sessions *fakeSessions
	states   *fakeStates
	prefs    *fakePrefs
	ran      []string
}

func newFixture(t *testing.T, commands ...Command) *fixture {
	t.Helper()
	api := newFakeAPI(t)
	sessions := newFakeSessions()
	states := &fakeStates{}
	prefs := &fakePrefs{locale: "uk"}
	f := &fixture{api: api, sessions: sessions, states: states, prefs: prefs}

	if len(commands) == 0 {
		commands = []Command{{
			Name: "ping",
			Help: "answer pong",
			Run: func(ctx context.Context, c *Context) error {
				f.ran = append(f.ran, "ping")
				return c.Reply(ctx, "pong for "+c.UserID())
			},
		}}
	}

	f.bot = New(Options{
		Name:     "finance",
		Intro:    "test bot",
		API:      api.client(t),
		Sessions: sessions,
		States:   states,
		Prefs:    prefs,
		Commands: commands,
		Callbacks: []Callback{{Prefix: "ping", Run: func(ctx context.Context, c *Context) error {
			f.ran = append(f.ran, "cb:"+c.Payload())
			return c.Show(ctx, "pong panel", Keyboard{Row(Data("‹ Menu", "home"))})
		}}},
		Home: func(ctx context.Context, c *Context) (string, Keyboard, error) {
			return "home panel", Keyboard{Row(Data("Ping", "ping:1"))}, nil
		},
	})
	return f
}

func TestConnectLearnsTheUsername(t *testing.T) {
	f := newFixture(t)

	if err := f.bot.Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if f.bot.Username() != "fm_test_bot" {
		t.Fatalf("username = %q", f.bot.Username())
	}
}

func TestUnknownCommandPointsAtHelp(t *testing.T) {
	f := newFixture(t)

	f.bot.Handle(context.Background(), message("/nonsense"))

	contains(t, f.api.last(t), i18n.T(i18n.Default, i18n.Unknown))
}

func TestHelpListsTheMenuCommand(t *testing.T) {
	f := newFixture(t)

	f.bot.Handle(context.Background(), message("/help"))

	contains(t, f.api.last(t), "/menu")
}

func TestHelpListsEveryCommand(t *testing.T) {
	f := newFixture(t)

	f.bot.Handle(context.Background(), message("/help"))

	reply := f.api.last(t)
	for _, want := range []string{"/ping", "/menu", "/unlink", "test bot"} {
		contains(t, reply, want)
	}
	if strings.Contains(reply, "/start") {
		t.Fatalf("help advertises /start, which only arrives through a deep link: %q", reply)
	}
}

func TestPrivateCommandRequiresALink(t *testing.T) {
	f := newFixture(t)

	f.bot.Handle(context.Background(), message("/ping"))

	contains(t, f.api.last(t), i18n.T(i18n.Default, i18n.LinkTitle))
	if len(f.ran) != 0 {
		t.Fatal("the command ran without a session")
	}
}

func TestLinkedCommandSeesTheSession(t *testing.T) {
	f := newFixture(t)
	f.sessions.linked[42] = &session.Session{UserID: "user-1", AccessToken: "access-1"}

	f.bot.Handle(context.Background(), message("/ping"))

	contains(t, f.api.last(t), "pong for user-1")
}

func TestStartWithAPayloadRedeemsIt(t *testing.T) {
	f := newFixture(t)

	f.bot.Handle(context.Background(), message("/start link-token-abc"))

	if len(f.sessions.redeemed) != 1 || f.sessions.redeemed[0] != "link-token-abc" {
		t.Fatalf("redeemed = %v", f.sessions.redeemed)
	}
	contains(t, f.api.last(t), i18n.T(i18n.Default, i18n.LinkedTitle))
}

func TestStartLeavesTheChatSignedInForTheHomePanel(t *testing.T) {
	api := newFakeAPI(t)
	sessions := newFakeSessions()
	var sawToken string

	b := New(Options{
		Name: "finance", Intro: "test bot", API: api.client(t), Sessions: sessions,
		States: &fakeStates{},
		Home: func(ctx context.Context, c *Context) (string, Keyboard, error) {
			if c.Session() == nil {
				return "", nil, errBoom
			}
			sawToken = c.Session().AccessToken
			return "home panel", Keyboard{Row(Data("Ping", "ping:1"))}, nil
		},
	})

	b.Handle(context.Background(), message("/start link-token-abc"))

	reply := api.last(t)
	contains(t, reply, i18n.T(i18n.Default, i18n.LinkedTitle))
	contains(t, reply, "home panel")
	if sawToken != "access-1" {
		t.Fatalf("home panel ran without the freshly redeemed session (token %q)", sawToken)
	}
}

func TestStartWhenAlreadyLinkedShowsTheHomePanel(t *testing.T) {
	f := newFixture(t)
	f.sessions.linked[42] = &session.Session{UserID: "user-1", AccessToken: "access-1"}

	f.bot.Handle(context.Background(), message("/start"))

	reply := f.api.last(t)
	contains(t, reply, i18n.T(i18n.Default, i18n.LinkedAlready))
	contains(t, reply, "home panel")
}

func TestStartWithAStaleTokenAsksForAFreshOne(t *testing.T) {
	f := newFixture(t)
	f.sessions.err = session.ErrLinkAgain

	f.bot.Handle(context.Background(), message("/start stale"))

	contains(t, f.api.last(t), i18n.T(i18n.Default, i18n.LinkSpentAgain))
}

func TestUnlinkRevokesTheSession(t *testing.T) {
	f := newFixture(t)
	f.sessions.linked[42] = &session.Session{UserID: "user-1"}

	f.bot.Handle(context.Background(), message("/unlink"))

	if len(f.sessions.unlinked) != 1 {
		t.Fatal("unlink did not reach the store")
	}
	contains(t, f.api.last(t), i18n.T(i18n.Default, i18n.Unlinked))
}

func TestUnlinkWithoutALinkIsHarmless(t *testing.T) {
	f := newFixture(t)

	f.bot.Handle(context.Background(), message("/unlink"))

	contains(t, f.api.last(t), i18n.T(i18n.Default, i18n.UnlinkNothing))
}

func TestCommandNameIgnoresTheBotSuffix(t *testing.T) {
	f := newFixture(t)
	f.sessions.linked[42] = &session.Session{UserID: "user-1"}

	f.bot.Handle(context.Background(), message("/ping@fm_test_bot"))

	contains(t, f.api.last(t), "pong")
}

func TestInputErrorsAreShownToTheUser(t *testing.T) {
	f := newFixture(t, Command{
		Name:   "boom",
		Help:   "fail on purpose",
		Public: true,
		Run: func(context.Context, *Context) error {
			return Invalid("I need an amount: /boom 12")
		},
	})

	f.bot.Handle(context.Background(), message("/boom"))

	contains(t, f.api.last(t), "I need an amount")
}

func TestInternalErrorsStayOpaque(t *testing.T) {
	f := newFixture(t, Command{
		Name:   "boom",
		Help:   "fail on purpose",
		Public: true,
		Run:    func(context.Context, *Context) error { return errBoom },
	})

	f.bot.Handle(context.Background(), message("/boom"))

	reply := f.api.last(t)
	contains(t, reply, i18n.T(i18n.Default, i18n.Failed))
	if got := reply; got == errBoom.Error() {
		t.Fatal("the raw error reached the user")
	}
}

func TestConnectPublishesTheCommandMenu(t *testing.T) {
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
	for _, want := range []string{"ping", "help", "menu", "unlink"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("command menu %q is missing %q", joined, want)
		}
	}
	if strings.Contains(joined, "start") {
		t.Fatalf("hidden command leaked into the menu: %q", joined)
	}
}

func TestMenuRendersTheHomePanel(t *testing.T) {
	f := newFixture(t)
	f.sessions.linked[42] = &session.Session{UserID: "user-1"}

	f.bot.Handle(context.Background(), message("/menu"))

	msg := f.api.lastMessage(t)
	contains(t, msg.Text, "home panel")
	if msg.ParseMode != "HTML" {
		t.Fatalf("parse mode = %q, want HTML", msg.ParseMode)
	}
	if got := f.api.buttons(t); len(got) != 1 || got[0] != "Ping=ping:1" {
		t.Fatalf("buttons = %v", got)
	}
}

func TestCallbackEditsTheMessageInPlace(t *testing.T) {
	f := newFixture(t)
	f.sessions.linked[42] = &session.Session{UserID: "user-1"}

	f.bot.Handle(context.Background(), callback("ping:7"))

	msg := f.api.lastMessage(t)
	if !msg.edited {
		t.Fatal("the callback sent a new message instead of editing")
	}
	if msg.MessageID != 555 {
		t.Fatalf("edited message id = %d, want 555", msg.MessageID)
	}
	contains(t, msg.Text, "pong panel")
	if len(f.ran) != 1 || f.ran[0] != "cb:7" {
		t.Fatalf("handler saw %v, want the payload 7", f.ran)
	}
}

func TestCallbackIsAlwaysAnswered(t *testing.T) {
	f := newFixture(t)
	f.sessions.linked[42] = &session.Session{UserID: "user-1"}

	f.bot.Handle(context.Background(), callback("ping:1"))

	f.api.mu.Lock()
	defer f.api.mu.Unlock()
	if len(f.api.toasts) == 0 {
		t.Fatal("the spinner was left running: no answerCallbackQuery")
	}
}

func TestStaleCallbackIsExplained(t *testing.T) {
	f := newFixture(t)
	f.sessions.linked[42] = &session.Session{UserID: "user-1"}

	f.bot.Handle(context.Background(), callback("gone:1"))

	if got := f.api.lastToast(t); !strings.Contains(got, i18n.T(i18n.Default, i18n.StaleButton)) {
		t.Fatalf("toast = %q", got)
	}
}

func TestCallbackFromAnUnlinkedChatIsRefused(t *testing.T) {
	f := newFixture(t)

	f.bot.Handle(context.Background(), callback("ping:1"))

	contains(t, f.api.last(t), i18n.T(i18n.Default, i18n.LinkTitle))
	if len(f.ran) != 0 {
		t.Fatal("the callback ran without a session")
	}
}

func TestFreeTextGoesToTheTextHandler(t *testing.T) {
	api := newFakeAPI(t)
	sessions := newFakeSessions()
	sessions.linked[42] = &session.Session{UserID: "user-1"}
	var seen string

	b := New(Options{
		Name: "finance", Intro: "test bot", API: api.client(t), Sessions: sessions,
		States: &fakeStates{},
		OnText: func(ctx context.Context, c *Context) error {
			seen = c.Args
			return c.Reply(ctx, "got it")
		},
	})

	b.Handle(context.Background(), message("250 groceries"))

	if seen != "250 groceries" {
		t.Fatalf("text handler saw %q", seen)
	}
	contains(t, api.last(t), "got it")
}

func TestStateIsStoredAndCleared(t *testing.T) {
	f := newFixture(t, Command{
		Name: "arm", Help: "arm a flow", Public: true,
		Run: func(ctx context.Context, c *Context) error {
			if err := c.SetState(ctx, "await", map[string]string{"category": "Groceries"}); err != nil {
				return err
			}
			return c.Reply(ctx, "armed")
		},
	})

	f.bot.Handle(context.Background(), message("/arm"))
	if f.states.kind != "await" || f.states.payload["category"] != "Groceries" {
		t.Fatalf("state = %q %v", f.states.kind, f.states.payload)
	}

	f.sessions.linked[42] = &session.Session{UserID: "user-1"}
	f.bot.Handle(context.Background(), message("/menu"))
	if f.states.kind != "" {
		t.Fatal("running a command left the pending flow armed")
	}
}

func TestHTMLIsEscapedInHelp(t *testing.T) {
	api := newFakeAPI(t)
	b := New(Options{
		Name: "finance", Intro: "bread & butter <budget>", API: api.client(t),
		Sessions: newFakeSessions(), States: &fakeStates{},
	})

	b.Handle(context.Background(), message("/help"))

	text := api.last(t)
	if strings.Contains(text, "<budget>") {
		t.Fatalf("unescaped angle brackets reached telegram: %q", text)
	}
	contains(t, text, "&amp; butter &lt;budget&gt;")
}

func TestBotsIgnoreOtherBotsAndEmptyText(t *testing.T) {
	f := newFixture(t)

	update := message("/ping")
	update.Message.From.IsBot = true
	f.bot.Handle(context.Background(), update)

	f.bot.Handle(context.Background(), message("   "))
	f.bot.Handle(context.Background(), telegram.Update{UpdateID: 7})

	if texts := f.api.texts(); len(texts) != 0 {
		t.Fatalf("sent %v, want nothing", texts)
	}
}

func TestLongRepliesAreSplit(t *testing.T) {
	long := make([]byte, 0, maxMessageRunes*2)
	for i := 0; i < maxMessageRunes*2; i++ {
		long = append(long, 'a')
	}

	f := newFixture(t, Command{
		Name:   "long",
		Help:   "send a wall of text",
		Public: true,
		Run: func(ctx context.Context, c *Context) error {
			return c.Reply(ctx, string(long))
		},
	})

	f.bot.Handle(context.Background(), message("/long"))

	if got := len(f.api.texts()); got < 2 {
		t.Fatalf("sent %d messages, want the text split", got)
	}
}
