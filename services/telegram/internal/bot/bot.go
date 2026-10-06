package bot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/nnc/family-manager/services/telegram/internal/i18n"
	"github.com/nnc/family-manager/services/telegram/internal/session"
	"github.com/nnc/family-manager/services/telegram/internal/telegram"
)

const (
	maxMessageRunes = 3500
	defaultTimeout  = 15 * time.Second
)

type Handler func(ctx context.Context, c *Context) error

type Sessions interface {
	Session(ctx context.Context, telegramUserID int64) (*session.Session, error)
	Redeem(ctx context.Context, linkToken string, from telegram.User, chatID int64) (*session.Session, error)
	Unlink(ctx context.Context, telegramUserID int64) error
	ExpireAccess(ctx context.Context, telegramUserID int64) error
	ApproveLogin(ctx context.Context, telegramUserID int64, userCode string) error
	DenyLogin(ctx context.Context, telegramUserID int64, userCode string) error
}

type Command struct {
	Name   string
	Args   string
	Help   string
	Public bool
	Hidden bool
	Run    Handler
}

type Callback struct {
	Prefix string
	Run    Handler
}

type Bot struct {
	name     string
	intro    string
	api      *telegram.Client
	sessions Sessions
	states   States
	prefs    Preferences
	log      *slog.Logger
	timeout  time.Duration

	username  string
	commands  map[string]*Command
	order     []*Command
	callbacks map[string]*Callback
	onText    Handler
	home      Home
}

type Home func(ctx context.Context, c *Context) (string, Keyboard, error)

type Options struct {
	Name      string
	Intro     string
	API       *telegram.Client
	Sessions  Sessions
	States    States
	Prefs     Preferences
	Log       *slog.Logger
	Timeout   time.Duration
	Commands  []Command
	Callbacks []Callback
	OnText    Handler
	Home      Home
}

func New(opts Options) *Bot {
	b := &Bot{
		name:      opts.Name,
		intro:     opts.Intro,
		api:       opts.API,
		sessions:  opts.Sessions,
		states:    opts.States,
		prefs:     opts.Prefs,
		log:       opts.Log,
		timeout:   opts.Timeout,
		commands:  map[string]*Command{},
		callbacks: map[string]*Callback{},
		onText:    opts.OnText,
		home:      opts.Home,
	}
	if b.log == nil {
		b.log = slog.Default()
	}
	if b.timeout == 0 {
		b.timeout = defaultTimeout
	}
	b.log = b.log.With(slog.String("bot", b.name))

	for _, c := range opts.Commands {
		b.register(c)
	}
	for _, c := range builtins() {
		b.register(c)
	}
	if b.prefs != nil {
		b.register(settingsCommand())
	}
	if b.home != nil {
		b.register(Command{
			Name: "menu",
			Help: string(i18n.HelpMenu),
			Run: func(ctx context.Context, c *Context) error {
				return b.showHome(ctx, c)
			},
		})
	}
	sort.Slice(b.order, func(i, j int) bool { return b.order[i].Name < b.order[j].Name })

	for _, cb := range opts.Callbacks {
		entry := cb
		b.callbacks[cb.Prefix] = &entry
	}
	for _, cb := range builtinCallbacks() {
		entry := cb
		b.callbacks[cb.Prefix] = &entry
	}
	if b.prefs != nil {
		for _, cb := range settingsCallbacks() {
			entry := cb
			b.callbacks[cb.Prefix] = &entry
		}
	}
	if b.home != nil {
		b.callbacks["home"] = &Callback{Prefix: "home", Run: func(ctx context.Context, c *Context) error {
			c.ClearState(ctx)
			return b.showHome(ctx, c)
		}}
	}

	return b
}

func (b *Bot) showHome(ctx context.Context, c *Context) error {
	text, keyboard, err := b.home(ctx, c)
	if err != nil {
		return err
	}
	return c.Show(ctx, text, keyboard)
}

func (b *Bot) register(c Command) {
	cmd := c
	b.commands[cmd.Name] = &cmd
	b.order = append(b.order, &cmd)
}

func (b *Bot) Name() string { return b.name }

func (b *Bot) API() *telegram.Client { return b.api }

func (b *Bot) Username() string { return b.username }

func (b *Bot) Connect(ctx context.Context) error {
	me, err := b.api.GetMe(ctx)
	if err != nil {
		return fmt.Errorf("bot %s: getMe: %w", b.name, err)
	}
	b.username = me.Username

	if err := b.api.SetMyCommands(ctx, b.menuCommands()); err != nil {
		b.log.WarnContext(ctx, "set command menu", slog.String("error", err.Error()))
	}

	b.log.InfoContext(ctx, "bot ready", slog.String("username", me.Username))
	return nil
}

func (b *Bot) menuCommands() []telegram.BotCommand {
	out := make([]telegram.BotCommand, 0, len(b.order))
	for _, c := range b.order {
		if c.Hidden {
			continue
		}
		out = append(out, telegram.BotCommand{
			Command:     c.Name,
			Description: i18n.T(i18n.Default, i18n.Key(c.Help)),
		})
	}
	return out
}

func (b *Bot) Handle(parent context.Context, update telegram.Update) {
	ctx, cancel := context.WithTimeout(parent, b.timeout)
	defer cancel()

	if q := update.CallbackQuery; q != nil {
		b.handleCallback(ctx, q)
		return
	}

	msg := update.Message
	if msg == nil {
		msg = update.EditedMessage
	}
	if msg == nil || msg.From == nil || msg.From.IsBot || strings.TrimSpace(msg.Text) == "" {
		return
	}

	b.handleMessage(ctx, msg)
}

func (b *Bot) handleMessage(ctx context.Context, msg *telegram.Message) {
	name, args := b.parse(msg.Text)
	c := &Context{
		Bot:      b,
		Chat:     msg.Chat.ID,
		ChatType: msg.Chat.Type,
		From:     *msg.From,
		Command:  name,
		Args:     args,
	}

	log := b.log.With(
		slog.String("command", name),
		slog.Int64("telegram_user_id", msg.From.ID),
	)

	if name == "" {
		b.handleFreeText(ctx, log, c)
		return
	}

	cmd, ok := b.commands[name]
	if !ok {
		b.say(ctx, log, c, c.T(i18n.Unknown))
		return
	}

	if cmd.Public {
		b.attachLocale(ctx, c)
	} else if !b.attachSession(ctx, log, c) {
		return
	}

	c.ClearState(ctx)
	b.run(ctx, log, c, cmd.Run)
}

func (b *Bot) handleFreeText(ctx context.Context, log *slog.Logger, c *Context) {
	if b.onText == nil {
		b.say(ctx, log, c, c.T(i18n.OnlyButtons))
		return
	}
	if !b.attachSession(ctx, log, c) {
		return
	}
	b.run(ctx, log, c, b.onText)
}

func (b *Bot) handleCallback(ctx context.Context, q *telegram.CallbackQuery) {
	c := &Context{
		Bot:        b,
		From:       q.From,
		Query:      q.Data,
		callbackID: q.ID,
		callback:   true,
	}
	if q.Message != nil {
		c.Chat = q.Message.Chat.ID
		c.ChatType = q.Message.Chat.Type
		c.MessageID = q.Message.MessageID
	}

	log := b.log.With(
		slog.String("callback", q.Data),
		slog.Int64("telegram_user_id", q.From.ID),
	)

	defer func() {
		if !c.answered {
			if err := b.api.AnswerCallbackQuery(ctx, q.ID, "", false); err != nil {
				log.WarnContext(ctx, "answer callback", slog.String("error", err.Error()))
			}
		}
	}()

	prefix, _, _ := strings.Cut(q.Data, ":")
	cb, ok := b.callbacks[prefix]
	if !ok {
		_ = c.Toast(ctx, c.T(i18n.StaleButton))
		return
	}

	if !b.attachSession(ctx, log, c) {
		return
	}
	b.run(ctx, log, c, cb.Run)
}

func (b *Bot) attachLocale(ctx context.Context, c *Context) {
	if c.session != nil {
		return
	}
	s, err := b.sessions.Session(ctx, c.From.ID)
	if err != nil {
		return
	}
	c.session = s
	c.locale = i18n.Match(s.Locale)
}

func (b *Bot) attachSession(ctx context.Context, log *slog.Logger, c *Context) bool {
	s, err := b.sessions.Session(ctx, c.From.ID)
	switch {
	case errors.Is(err, session.ErrNotLinked), errors.Is(err, session.ErrLinkAgain):
		b.say(ctx, log, c, b.linkPrompt(c.Locale()))
		return false
	case err != nil:
		log.ErrorContext(ctx, "resolve session", slog.String("error", err.Error()))
		b.say(ctx, log, c, c.T(i18n.Internal))
		return false
	}
	c.session = s
	c.locale = i18n.Match(s.Locale)
	return true
}

func (b *Bot) run(ctx context.Context, log *slog.Logger, c *Context, run Handler) {
	err := run(ctx, c)
	if isNoFamily(err) && c.Command != "start" && b.renewSession(ctx, log, c) {
		err = run(ctx, c)
	}
	if err != nil {
		log.ErrorContext(ctx, "handler failed", slog.String("error", err.Error()))
		b.say(ctx, log, c, friendly(c.Locale(), err))
	}
}

func (b *Bot) renewSession(ctx context.Context, log *slog.Logger, c *Context) bool {
	if err := b.sessions.ExpireAccess(ctx, c.From.ID); err != nil {
		log.WarnContext(ctx, "expire access after a no-family error", slog.String("error", err.Error()))
		return false
	}
	s, err := b.sessions.Session(ctx, c.From.ID)
	if err != nil {
		log.WarnContext(ctx, "renew session after a no-family error", slog.String("error", err.Error()))
		return false
	}
	c.session = s
	return true
}

func isNoFamily(err error) bool {
	return connect.CodeOf(err) == connect.CodeFailedPrecondition && strings.Contains(err.Error(), "belongs to no family")
}

func precondition(err error) (i18n.Key, bool) {
	switch {
	case isNoFamily(err):
		return i18n.NoFamily, true
	case connect.CodeOf(err) == connect.CodeFailedPrecondition && strings.Contains(err.Error(), "not set up yet"):
		return i18n.NotSetUp, true
	}
	return "", false
}

func (b *Bot) parse(text string) (name, args string) {
	trimmed := strings.TrimSpace(text)
	head, rest, _ := strings.Cut(trimmed, " ")
	if !strings.HasPrefix(head, "/") {
		return "", trimmed
	}
	name = strings.ToLower(strings.TrimPrefix(head, "/"))
	if at, _, found := strings.Cut(name, "@"); found {
		name = at
	}
	return name, strings.TrimSpace(rest)
}

func (b *Bot) say(ctx context.Context, log *slog.Logger, c *Context, text string) {
	if err := c.Reply(ctx, text); err != nil {
		log.ErrorContext(ctx, "send message", slog.String("error", err.Error()))
	}
}

func (b *Bot) linkPrompt(locale i18n.Locale) string {
	return Lines(
		Bold(i18n.T(locale, i18n.LinkTitle)),
		"",
		Esc(i18n.T(locale, i18n.LinkBody)),
	)
}

func (b *Bot) helpText(locale i18n.Locale) string {
	var sb strings.Builder
	sb.WriteString(Bold(i18n.T(locale, i18n.Key(b.intro))) + "\n\n")
	for _, c := range b.order {
		if c.Hidden {
			continue
		}
		sb.WriteString("/" + c.Name)
		if c.Args != "" {
			sb.WriteString(" " + Italic(c.Args))
		}
		sb.WriteString(" — " + Esc(i18n.T(locale, i18n.Key(c.Help))) + "\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

func friendly(locale i18n.Locale, err error) string {
	var invalid *InputError
	if errors.As(err, &invalid) {
		return invalid.Error()
	}
	if key, ok := precondition(err); ok {
		return i18n.T(locale, key)
	}
	return i18n.T(locale, i18n.Failed)
}

type InputError struct{ Msg string }

func (e *InputError) Error() string { return e.Msg }

func Invalid(format string, args ...any) error {
	return &InputError{Msg: fmt.Sprintf(format, args...)}
}

func timestamp(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}
