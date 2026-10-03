package bot

import (
	"context"
	"testing"

	"github.com/nnc/family-manager/services/telegram/internal/i18n"
	"github.com/nnc/family-manager/services/telegram/internal/session"
)

func (f *fixture) linkAda() {
	f.sessions.linked[42] = &session.Session{UserID: "user-1", AccessToken: "access-1", Locale: "en"}
}

func TestStartLoginAsksForConfirmation(t *testing.T) {
	f := newFixture(t)
	f.linkAda()

	f.bot.Handle(context.Background(), message("/start login_bcdfghjk"))

	contains(t, f.api.last(t), i18n.T(i18n.EN, i18n.LoginConfirm))
	contains(t, f.api.last(t), "BCDF-GHJK")
	buttons := f.api.buttons(t)
	want := []string{
		i18n.T(i18n.EN, i18n.LoginApprove) + "=login:ok:BCDFGHJK",
		i18n.T(i18n.EN, i18n.LoginDeny) + "=login:no:BCDFGHJK",
	}
	if len(buttons) != 2 || buttons[0] != want[0] || buttons[1] != want[1] {
		t.Fatalf("buttons = %v, want %v", buttons, want)
	}
	if len(f.sessions.approved) != 0 || len(f.sessions.redeemed) != 0 {
		t.Fatal("the deep link alone approved or redeemed something")
	}
}

func TestStartLoginFromAnUnlinkedAccountSaysLinkFirst(t *testing.T) {
	f := newFixture(t)

	f.bot.Handle(context.Background(), message("/start login_BCDFGHJK"))

	contains(t, f.api.last(t), i18n.T(i18n.Default, i18n.LoginNotLinked))
	if len(f.sessions.redeemed) != 0 {
		t.Fatal("a login payload was redeemed as a link token")
	}
}

func TestStartLoginWithAMalformedCodeIsExpired(t *testing.T) {
	f := newFixture(t)
	f.linkAda()

	f.bot.Handle(context.Background(), message("/start login_short"))

	contains(t, f.api.last(t), i18n.T(i18n.EN, i18n.LoginExpired))
	if f.api.lastMessage(t).ReplyMarkup != nil {
		t.Fatal("a malformed code still offered buttons")
	}
}

func TestApproveButtonApprovesAsTheLinkedUser(t *testing.T) {
	f := newFixture(t)
	f.linkAda()

	f.bot.Handle(context.Background(), callback("login:ok:BCDFGHJK"))

	if len(f.sessions.approved) != 1 || f.sessions.approved[0] != "BCDFGHJK" {
		t.Fatalf("approved = %v", f.sessions.approved)
	}
	msg := f.api.lastMessage(t)
	if !msg.edited {
		t.Fatal("the confirmation was not replaced in place")
	}
	contains(t, msg.Text, i18n.T(i18n.EN, i18n.LoginApproved))
	if msg.ReplyMarkup != nil {
		t.Fatal("the decided prompt kept its buttons")
	}
}

func TestDenyButtonDenies(t *testing.T) {
	f := newFixture(t)
	f.linkAda()

	f.bot.Handle(context.Background(), callback("login:no:BCDFGHJK"))

	if len(f.sessions.denied) != 1 || len(f.sessions.approved) != 0 {
		t.Fatalf("denied = %v approved = %v", f.sessions.denied, f.sessions.approved)
	}
	contains(t, f.api.last(t), i18n.T(i18n.EN, i18n.LoginDenied))
}

func TestApproveAnExpiredCodeExplains(t *testing.T) {
	f := newFixture(t)
	f.linkAda()
	f.sessions.decide = session.ErrLoginExpired

	f.bot.Handle(context.Background(), callback("login:ok:BCDFGHJK"))

	contains(t, f.api.last(t), i18n.T(i18n.EN, i18n.LoginExpired))
}

func TestApproveWhenThrottledExplains(t *testing.T) {
	f := newFixture(t)
	f.linkAda()
	f.sessions.decide = session.ErrLoginThrottled

	f.bot.Handle(context.Background(), callback("login:ok:BCDFGHJK"))

	contains(t, f.api.last(t), i18n.T(i18n.EN, i18n.LoginThrottled))
}

func TestApproveFromAnUnlinkedChatIsRefused(t *testing.T) {
	f := newFixture(t)

	f.bot.Handle(context.Background(), callback("login:ok:BCDFGHJK"))

	if len(f.sessions.approved) != 0 {
		t.Fatal("an unlinked chat approved a login")
	}
	contains(t, f.api.last(t), i18n.T(i18n.Default, i18n.LinkTitle))
}

func TestStartWithALinkTokenStillRedeems(t *testing.T) {
	f := newFixture(t)

	f.bot.Handle(context.Background(), message("/start loginless-token"))

	if len(f.sessions.redeemed) != 1 || f.sessions.redeemed[0] != "loginless-token" {
		t.Fatalf("redeemed = %v", f.sessions.redeemed)
	}
}
