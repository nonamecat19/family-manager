package bot

import (
	"context"
	"strings"
	"testing"

	"github.com/nnc/family-manager/services/telegram/internal/i18n"
	"github.com/nnc/family-manager/services/telegram/internal/session"
)

func TestSettingsPanelShowsBothLanguages(t *testing.T) {
	f := newFixture(t)
	f.sessions.linked[42] = &session.Session{UserID: "user-1", AccessToken: "access-1", Locale: "uk"}

	f.bot.Handle(context.Background(), message("/settings"))

	msg := f.api.lastMessage(t)
	contains(t, msg.Text, i18n.T(i18n.UK, i18n.SettingsTitle))
	buttons := strings.Join(f.api.buttons(t), " ")
	for _, want := range []string{"lang:uk", "lang:en", "Українська", "English"} {
		if !strings.Contains(buttons, want) {
			t.Fatalf("buttons %q missing %q", buttons, want)
		}
	}
	if !strings.Contains(buttons, "✓ 🇺🇦") {
		t.Fatalf("the current language is not ticked: %q", buttons)
	}
}

func TestPickingALanguageSavesItAndRedrawsInThatLanguage(t *testing.T) {
	f := newFixture(t)
	f.sessions.linked[42] = &session.Session{UserID: "user-1", AccessToken: "access-1", Locale: "uk"}

	f.bot.Handle(context.Background(), callback("lang:en"))

	if len(f.prefs.saved) != 1 || f.prefs.saved[0] != "en" {
		t.Fatalf("saved = %v, want [en]", f.prefs.saved)
	}
	if len(f.sessions.expired) != 1 {
		t.Fatal("the cached access token was not expired, so the JWT keeps the old locale")
	}

	msg := f.api.lastMessage(t)
	contains(t, msg.Text, i18n.T(i18n.EN, i18n.SettingsTitle))
	contains(t, msg.Text, i18n.T(i18n.EN, i18n.SettingsShared))
	if got := f.api.lastToast(t); got != i18n.T(i18n.EN, i18n.SettingsSaved) {
		t.Fatalf("toast = %q, want the English confirmation", got)
	}
}

func TestBotCopyFollowsTheStoredLocale(t *testing.T) {
	f := newFixture(t)
	f.sessions.linked[42] = &session.Session{UserID: "user-1", AccessToken: "access-1", Locale: "en"}

	f.bot.Handle(context.Background(), message("/unlink"))

	contains(t, f.api.last(t), i18n.T(i18n.EN, i18n.Unlinked))
}

func TestUnlinkedChatsAreGreetedInTheDefaultLocale(t *testing.T) {
	f := newFixture(t)

	f.bot.Handle(context.Background(), message("/ping"))

	contains(t, f.api.last(t), i18n.T(i18n.Default, i18n.LinkTitle))
}

func TestSettingsIsAbsentWithoutAPreferencesStore(t *testing.T) {
	api := newFakeAPI(t)
	b := New(Options{
		Name: "finance", Intro: string(i18n.FinanceIntro), API: api.client(t),
		Sessions: newFakeSessions(), States: &fakeStates{},
	})

	b.Handle(context.Background(), message("/settings"))

	contains(t, api.last(t), i18n.T(i18n.Default, i18n.Unknown))
}
