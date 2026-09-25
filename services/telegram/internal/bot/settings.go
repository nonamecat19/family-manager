package bot

import (
	"context"
	"strings"

	"github.com/nnc/family-manager/services/telegram/internal/i18n"
)

type Preferences interface {
	Locale(ctx context.Context, accessToken string) (string, error)
	SetLocale(ctx context.Context, accessToken, locale string) (string, error)
}

func settingsCommand() Command {
	return Command{
		Name: "settings",
		Help: string(i18n.HelpSettings),
		Run:  showSettings,
	}
}

func settingsCallbacks() []Callback {
	return []Callback{
		{Prefix: "settings", Run: showSettings},
		{Prefix: "lang", Run: setLanguage},
	}
}

func showSettings(ctx context.Context, c *Context) error {
	current := c.Locale()
	if c.Bot.prefs != nil {
		if stored, err := c.Bot.prefs.Locale(ctx, c.session.AccessToken); err == nil && stored != "" {
			current = i18n.Match(stored)
		}
	}

	return c.Show(ctx, Lines(
		Bold(c.T(i18n.SettingsTitle)),
		"",
		Bold(c.T(i18n.SettingsBody))+"   "+current.Flag()+" "+Esc(current.Name()),
		Esc(c.T(i18n.SettingsLanguage)),
		"",
		Italic(c.T(i18n.SettingsShared)),
	), languageKeyboard(current, c.T(i18n.Menu)))
}

func languageKeyboard(current i18n.Locale, back string) Keyboard {
	var row []Button
	for _, locale := range i18n.Supported {
		label := locale.Flag() + " " + locale.Name()
		if locale == current {
			label = "✓ " + label
		}
		row = append(row, Data(label, "lang:"+string(locale)))
	}
	return Keyboard{row, Row(Data(back, "home"))}
}

func setLanguage(ctx context.Context, c *Context) error {
	if c.Bot.prefs == nil {
		return c.Toast(ctx, c.T(i18n.StaleButton))
	}

	chosen := i18n.Match(strings.TrimSpace(c.Payload()))
	saved, err := c.Bot.prefs.SetLocale(ctx, c.session.AccessToken, string(chosen))
	if err != nil {
		return err
	}
	c.locale = i18n.Match(saved)
	c.session.Locale = saved

	if err := c.Bot.sessions.ExpireAccess(ctx, c.Bot.name, c.From.ID); err != nil {
		c.Bot.log.WarnContext(ctx, "expire access after a locale change", "error", err.Error())
	}

	if err := c.Toast(ctx, c.T(i18n.SettingsSaved)); err != nil {
		return err
	}
	return showSettings(ctx, c)
}
