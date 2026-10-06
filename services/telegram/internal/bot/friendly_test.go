package bot

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"

	"github.com/nnc/family-manager/services/telegram/internal/i18n"
	"github.com/nnc/family-manager/services/telegram/internal/session"
)

func TestFriendlyExplainsMissingFamily(t *testing.T) {
	err := fmt.Errorf("balance: %w",
		connect.NewError(connect.CodeFailedPrecondition, errors.New("caller belongs to no family")))
	if got, want := friendly(i18n.EN, err), i18n.T(i18n.EN, i18n.NoFamily); got != want {
		t.Fatalf("friendly = %q, want %q", got, want)
	}
}

func TestFriendlyFallsBackForOtherPreconditions(t *testing.T) {
	err := connect.NewError(connect.CodeFailedPrecondition, errors.New("account archived"))
	if got, want := friendly(i18n.EN, err), i18n.T(i18n.EN, i18n.Failed); got != want {
		t.Fatalf("friendly = %q, want %q", got, want)
	}
}

func TestFriendlyKeepsInputErrors(t *testing.T) {
	if got := friendly(i18n.EN, Invalid("amount %q is not a number", "x")); got != `amount "x" is not a number` {
		t.Fatalf("friendly = %q", got)
	}
}

func noFamily() error {
	return connect.NewError(connect.CodeFailedPrecondition, errors.New("caller belongs to no family"))
}

func TestNoFamilyRenewsTheTokenAndRetriesOnce(t *testing.T) {
	api := newFakeAPI(t)
	sessions := newFakeSessions()
	sessions.linked[42] = &session.Session{UserID: "user-1", AccessToken: "access-1"}
	calls := 0
	b := New(Options{
		Name: "finance", Intro: "test bot", API: api.client(t), Sessions: sessions, States: &fakeStates{},
		Commands: []Command{{Name: "balance", Help: "balance", Run: func(ctx context.Context, c *Context) error {
			calls++
			if calls == 1 {
				return noFamily()
			}
			return c.Reply(ctx, "balance ok")
		}}},
	})

	b.Handle(context.Background(), message("/balance"))

	if calls != 2 || len(sessions.expired) != 1 {
		t.Fatalf("calls = %d, expired = %v; want one retry after expiring the token", calls, sessions.expired)
	}
	contains(t, api.last(t), "balance ok")
}

func TestNoFamilyAfterRetryExplainsWhatToDo(t *testing.T) {
	api := newFakeAPI(t)
	sessions := newFakeSessions()
	sessions.linked[42] = &session.Session{UserID: "user-1", AccessToken: "access-1"}
	b := New(Options{
		Name: "finance", Intro: "test bot", API: api.client(t), Sessions: sessions, States: &fakeStates{},
		Commands: []Command{{Name: "balance", Help: "balance", Run: func(context.Context, *Context) error {
			return noFamily()
		}}},
	})

	b.Handle(context.Background(), message("/balance"))

	contains(t, api.last(t), i18n.T(i18n.Default, i18n.NoFamily))
}

func TestLinkingWithoutAFamilyConfirmsTheLinkAndExplains(t *testing.T) {
	api := newFakeAPI(t)
	sessions := newFakeSessions()
	b := New(Options{
		Name: "finance", Intro: "test bot", API: api.client(t), Sessions: sessions, States: &fakeStates{},
		Home: func(context.Context, *Context) (string, Keyboard, error) { return "", nil, noFamily() },
	})

	b.Handle(context.Background(), message("/start link-token-abc"))

	reply := api.last(t)
	contains(t, reply, i18n.T(i18n.Default, i18n.LinkedTitle))
	contains(t, reply, Esc(i18n.T(i18n.Default, i18n.NoFamily)))
	if len(sessions.redeemed) != 1 {
		t.Fatalf("redeemed %v; the link token must be spent exactly once", sessions.redeemed)
	}
}

func TestFriendlyExplainsUnfinishedSetup(t *testing.T) {
	err := connect.NewError(connect.CodeFailedPrecondition, errors.New("household finance is not set up yet"))
	if got, want := friendly(i18n.EN, err), i18n.T(i18n.EN, i18n.NotSetUp); got != want {
		t.Fatalf("friendly = %q, want %q", got, want)
	}
}
