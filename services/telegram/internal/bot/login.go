package bot

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/nnc/family-manager/services/telegram/internal/i18n"
	"github.com/nnc/family-manager/services/telegram/internal/session"
)

const (
	loginPayloadPrefix = "login_"
	loginCallback      = "login"
	loginApprove       = "ok"
	loginDeny          = "no"
	privateChat        = "private"
)

var loginCodeShape = regexp.MustCompile(`^[A-Za-z0-9]{8}$`)

func confirmLogin(ctx context.Context, c *Context, code string) error {
	if !inPrivateChat(c) {
		return c.Reply(ctx, Esc(c.T(i18n.LoginPrivateOnly)))
	}
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
	owner := strconv.FormatInt(c.From.ID, 10)

	return c.Send(ctx, Lines(
		Bold(c.T(i18n.LoginConfirm)),
		"",
		Esc(c.T(i18n.LoginConfirmBody, displayLoginCode(code))),
	), Keyboard{Row(
		Data(c.T(i18n.LoginApprove), loginButton(loginApprove, code, owner)),
		Data(c.T(i18n.LoginDeny), loginButton(loginDeny, code, owner)),
	)})
}

func decideLogin(ctx context.Context, c *Context) error {
	if !inPrivateChat(c) {
		return c.Alert(ctx, c.T(i18n.LoginPrivateOnly))
	}
	action, code, owner := c.PayloadAt(0), c.PayloadAt(1), c.PayloadAt(2)
	if owner != strconv.FormatInt(c.From.ID, 10) {
		return c.Toast(ctx, c.T(i18n.StaleButton))
	}
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

func inPrivateChat(c *Context) bool {
	return c.ChatType == privateChat && c.Chat == c.From.ID
}

func loginButton(action, code, owner string) string {
	return loginCallback + ":" + action + ":" + code + ":" + owner
}

func loginExpired(c *Context) string {
	return Lines(Bold(c.T(i18n.LoginExpired)), "", Esc(c.T(i18n.LoginExpiredBody)))
}

func displayLoginCode(code string) string {
	half := len(code) / 2
	return code[:half] + "-" + code[half:]
}
