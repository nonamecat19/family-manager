package bot

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"

	"github.com/nnc/family-manager/services/telegram/internal/i18n"
	"github.com/nnc/family-manager/services/telegram/internal/session"
	"github.com/nnc/family-manager/services/telegram/internal/telegram"
)

type Context struct {
	Bot       *Bot
	Chat      int64
	ChatType  string
	MessageID int64
	From      telegram.User
	Command   string
	Args      string
	Query     string

	session    *session.Session
	locale     i18n.Locale
	callbackID string
	callback   bool
	answered   bool
}

func (c *Context) Session() *session.Session { return c.session }

func (c *Context) Locale() i18n.Locale {
	if c.locale == "" {
		return i18n.Default
	}
	return c.locale
}

func (c *Context) T(key i18n.Key, args ...any) string {
	return i18n.T(c.Locale(), key, args...)
}

func (c *Context) IsCallback() bool { return c.callback }

func (c *Context) UserID() string {
	if c.session == nil {
		return ""
	}
	return c.session.UserID
}

func (c *Context) Authorize(req connect.AnyRequest) connect.AnyRequest {
	if c.session != nil {
		req.Header().Set("Authorization", "Bearer "+c.session.AccessToken)
	}
	return req
}

func (c *Context) Payload() string {
	_, rest, _ := strings.Cut(c.Query, ":")
	return rest
}

func (c *Context) PayloadAt(n int) string {
	parts := strings.Split(c.Payload(), ":")
	if n >= len(parts) {
		return ""
	}
	return parts[n]
}

func (c *Context) Reply(ctx context.Context, text string) error {
	return c.Send(ctx, text, nil)
}

func (c *Context) Replyf(ctx context.Context, format string, args ...any) error {
	return c.Reply(ctx, fmt.Sprintf(format, args...))
}

func (c *Context) Send(ctx context.Context, text string, keyboard Keyboard) error {
	chunks := split(text, maxMessageRunes)
	for i, chunk := range chunks {
		params := telegram.SendMessageParams{
			ChatID:             c.Chat,
			Text:               chunk,
			ParseMode:          "HTML",
			DisableLinkPreview: true,
		}
		if i == len(chunks)-1 {
			params.ReplyMarkup = keyboard.markup()
		}
		if _, err := c.Bot.api.SendMessage(ctx, params); err != nil {
			return err
		}
	}
	return nil
}

func (c *Context) Show(ctx context.Context, text string, keyboard Keyboard) error {
	if !c.callback || c.MessageID == 0 || len([]rune(text)) > maxMessageRunes {
		return c.Send(ctx, text, keyboard)
	}
	return c.Bot.api.EditMessageText(ctx, telegram.EditMessageTextParams{
		ChatID:             c.Chat,
		MessageID:          c.MessageID,
		Text:               text,
		ParseMode:          "HTML",
		DisableLinkPreview: true,
		ReplyMarkup:        keyboard.markup(),
	})
}

func (c *Context) Toast(ctx context.Context, text string) error {
	return c.answer(ctx, text, false)
}

func (c *Context) Alert(ctx context.Context, text string) error {
	return c.answer(ctx, text, true)
}

func (c *Context) answer(ctx context.Context, text string, alert bool) error {
	if !c.callback || c.answered {
		return nil
	}
	c.answered = true
	return c.Bot.api.AnswerCallbackQuery(ctx, c.callbackID, text, alert)
}

func (c *Context) Arg(n int) string {
	fields := strings.Fields(c.Args)
	if n >= len(fields) {
		return ""
	}
	return fields[n]
}

func split(text string, limit int) []string {
	runes := []rune(text)
	if len(runes) <= limit {
		return []string{text}
	}

	var chunks []string
	for len(runes) > limit {
		cut := limit
		if idx := strings.LastIndex(string(runes[:limit]), "\n"); idx > limit/2 {
			cut = len([]rune(string(runes[:limit])[:idx]))
		}
		chunks = append(chunks, strings.TrimRight(string(runes[:cut]), "\n"))
		runes = runes[cut:]
	}
	if trimmed := strings.TrimSpace(string(runes)); trimmed != "" {
		chunks = append(chunks, trimmed)
	}
	return chunks
}
