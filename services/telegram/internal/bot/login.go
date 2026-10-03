package bot

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/nnc/family-manager/services/telegram/internal/i18n"
	"github.com/nnc/family-manager/services/telegram/internal/session"
)

const (
	loginPayloadPrefix = "login_"
	loginCallback      = "login"
	loginApprove       = "ok"
	loginDeny          = "no"
)

var loginCodeShape = regexp.MustCompile(`^[A-Za-z0-9]{8}$`)

func confirmLogin(ctx context.Context, c *Context, code string) error {
	if _, err := c.Bot.sessions.Session(ctx, c.From.ID); err != nil {
		if errors.Is(err, session.ErrNotLinked) || errors.Is(err, session.ErrLinkAgain) {
			return c.Reply(ctx, Lines(
				Bold(c.T(i18n.LoginNotLinked)),
				"",
				Esc(c.T(i18n.LoginNotLinkedBody)),
			))
		}
		return err
	}

	if !loginCodeShape.MatchString(code) {
		return c.Reply(ctx, loginExpired(c))
	}
	code = strings.ToUpper(code)

	return c.Send(ctx, Lines(
		Bold(c.T(i18n.LoginConfirm)),
		"",
		Esc(c.T(i18n.LoginConfirmBody, displayLoginCode(code))),
	), Keyboard{Row(
		Data(c.T(i18n.LoginApprove), loginCallback+":"+loginApprove+":"+code),
		Data(c.T(i18n.LoginDeny), loginCallback+":"+loginDeny+":"+code),
	)})
}

func decideLogin(ctx context.Context, c *Context) error {
	action, code := c.PayloadAt(0), c.PayloadAt(1)
	if !loginCodeShape.MatchString(code) {
		return c.Show(ctx, loginExpired(c), nil)
	}

	var (
		err  error
		done string
	)
	switch action {
	case loginApprove:
		err = c.Bot.sessions.ApproveLogin(ctx, c.From.ID, code)
		done = Lines(Bold(c.T(i18n.LoginApproved)), "", Esc(c.T(i18n.LoginApprovedBody)))
	case loginDeny:
		err = c.Bot.sessions.DenyLogin(ctx, c.From.ID, code)
		done = Bold(c.T(i18n.LoginDenied))
	default:
		return c.Toast(ctx, c.T(i18n.StaleButton))
	}

	switch {
	case errors.Is(err, session.ErrLoginExpired):
		return c.Show(ctx, loginExpired(c), nil)
	case errors.Is(err, session.ErrLoginThrottled):
		return c.Show(ctx, Esc(c.T(i18n.LoginThrottled)), nil)
	case errors.Is(err, session.ErrNotLinked), errors.Is(err, session.ErrLinkAgain):
		return c.Show(ctx, c.Bot.linkPrompt(c.Locale()), nil)
	case err != nil:
		return err
	}
	return c.Show(ctx, done, nil)
}

func loginExpired(c *Context) string {
	return Lines(Bold(c.T(i18n.LoginExpired)), "", Esc(c.T(i18n.LoginExpiredBody)))
}

func displayLoginCode(code string) string {
	half := len(code) / 2
	return code[:half] + "-" + code[half:]
}
