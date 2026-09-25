package bot

import (
	"context"
	"errors"
	"strings"

	"github.com/nnc/family-manager/services/telegram/internal/i18n"
	"github.com/nnc/family-manager/services/telegram/internal/session"
)

func builtins() []Command {
	return []Command{
		{
			Name:   "start",
			Args:   "[link token]",
			Help:   string(i18n.HelpStart),
			Public: true,
			Hidden: true,
			Run:    start,
		},
		{
			Name:   "help",
			Help:   string(i18n.HelpHelp),
			Public: true,
			Run: func(ctx context.Context, c *Context) error {
				return c.Reply(ctx, c.Bot.helpText(c.Locale()))
			},
		},
		{
			Name:   "unlink",
			Help:   string(i18n.HelpUnlink),
			Public: true,
			Run:    unlink,
		},
	}
}

func builtinCallbacks() []Callback {
	return []Callback{
		{Prefix: "noop", Run: func(ctx context.Context, c *Context) error { return c.Toast(ctx, "") }},
	}
}

func start(ctx context.Context, c *Context) error {
	payload := strings.TrimSpace(c.Args)
	if payload == "" {
		if s, err := c.Bot.sessions.Session(ctx, c.Bot.name, c.From.ID); err == nil {
			c.session = s
			c.locale = i18n.Match(s.Locale)
			return c.Bot.welcome(ctx, c, c.T(i18n.LinkedAlready))
		}
		return c.Reply(ctx, c.Bot.linkPrompt(c.Locale()))
	}

	s, err := c.Bot.sessions.Redeem(ctx, c.Bot.name, payload, c.From, c.Chat)
	if err != nil {
		if errors.Is(err, session.ErrLinkAgain) {
			return c.Reply(ctx, Lines(
				Bold(c.T(i18n.LinkSpent)),
				"",
				Esc(c.T(i18n.LinkSpentAgain)),
			))
		}
		return err
	}
	c.session = s
	c.locale = i18n.Match(s.Locale)

	return c.Bot.welcome(ctx, c, c.T(i18n.LinkedTitle))
}

func (b *Bot) welcome(ctx context.Context, c *Context, headline string) error {
	if b.home == nil {
		return c.Reply(ctx, Lines(Bold(headline), "", b.helpText(c.Locale())))
	}
	text, keyboard, err := b.home(ctx, c)
	if err != nil {
		return err
	}
	return c.Send(ctx, Lines(Bold(headline), "", text), keyboard)
}

func unlink(ctx context.Context, c *Context) error {
	switch err := c.Bot.sessions.Unlink(ctx, c.Bot.name, c.From.ID); {
	case errors.Is(err, session.ErrNotLinked):
		return c.Reply(ctx, c.T(i18n.UnlinkNothing))
	case err != nil:
		return err
	}
	c.ClearState(ctx)
	return c.Reply(ctx, Lines(
		Bold(c.T(i18n.Unlinked)),
		"",
		Esc(c.T(i18n.UnlinkedBody)),
	))
}
