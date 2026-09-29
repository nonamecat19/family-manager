package session

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/nnc/family-manager/services/telegram/db"
	"github.com/nnc/family-manager/services/telegram/internal/telegram"
)

type fixture struct {
	store *Store
	q     *fakeQueries
	auth  *fakeAuth
	now   time.Time
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{q: newFakeQueries(), auth: &fakeAuth{}, now: time.Now()}
	f.store = NewStore(Options{
		Queries: f.q,
		Box:     newTestBox(t),
		Auth:    f.auth,
		Now:     func() time.Time { return f.now },
	})
	return f
}

var ada = telegram.User{ID: 42, FirstName: "Ada", Username: "ada"}

func (f *fixture) link(t *testing.T) *Session {
	t.Helper()
	s, err := f.store.Redeem(context.Background(), "finance", "link-token", ada, 99)
	if err != nil {
		t.Fatalf("Redeem: %v", err)
	}
	return s
}

func TestRedeemStoresTheSessionEncrypted(t *testing.T) {
	f := newFixture(t)

	s := f.link(t)

	if s.UserID != testUserID || s.AccessToken != "access-0" {
		t.Fatalf("session = %+v", s)
	}
	if len(f.auth.redeemed) != 1 || f.auth.redeemed[0].GetExternalId() != "42" {
		t.Fatalf("redeemed = %+v, want the telegram user id as external id", f.auth.redeemed)
	}

	link, err := f.q.GetLink(context.Background(), db.GetLinkParams{Bot: "finance", TelegramUserID: 42})
	if err != nil {
		t.Fatalf("GetLink: %v", err)
	}
	if bytes.Contains(link.RefreshToken, []byte("refresh-0")) ||
		bytes.Contains(link.AccessToken, []byte("access-0")) {
		t.Fatal("tokens are stored in the clear")
	}
	if link.ChatID != 99 || link.TelegramUsername != "ada" {
		t.Fatalf("link = %+v", link)
	}
}

func TestSessionReusesAFreshAccessToken(t *testing.T) {
	f := newFixture(t)
	f.link(t)

	s, err := f.store.Session(context.Background(), "finance", 42)
	if err != nil {
		t.Fatalf("Session: %v", err)
	}
	if s.AccessToken != "access-0" {
		t.Fatalf("access token = %q, want the stored one", s.AccessToken)
	}
	if len(f.auth.refreshed) != 0 {
		t.Fatal("a fresh token was refreshed anyway")
	}
}

func TestSessionRefreshesAnExpiredAccessToken(t *testing.T) {
	f := newFixture(t)
	f.link(t)

	f.now = f.now.Add(20 * time.Minute)

	s, err := f.store.Session(context.Background(), "finance", 42)
	if err != nil {
		t.Fatalf("Session: %v", err)
	}
	if s.AccessToken != "access-1" {
		t.Fatalf("access token = %q, want a refreshed one", s.AccessToken)
	}
	if len(f.auth.refreshed) != 1 || f.auth.refreshed[0] != "refresh-0" {
		t.Fatalf("refreshed = %v", f.auth.refreshed)
	}

	again, err := f.store.Session(context.Background(), "finance", 42)
	if err != nil {
		t.Fatalf("Session: %v", err)
	}
	if again.AccessToken != "access-1" {
		t.Fatalf("the rotated token was not stored: %q", again.AccessToken)
	}
}

func TestSessionDropsALinkTheAuthServiceRejects(t *testing.T) {
	f := newFixture(t)
	f.link(t)
	f.now = f.now.Add(20 * time.Minute)
	f.auth.refreshErr = connect.NewError(connect.CodeUnauthenticated, errors.New("revoked"))

	_, err := f.store.Session(context.Background(), "finance", 42)
	if !errors.Is(err, ErrLinkAgain) {
		t.Fatalf("err = %v, want ErrLinkAgain", err)
	}
	if len(f.q.deleted) != 1 {
		t.Fatal("the dead link was kept")
	}
}

func TestSessionWithoutALink(t *testing.T) {
	f := newFixture(t)

	_, err := f.store.Session(context.Background(), "finance", 42)
	if !errors.Is(err, ErrNotLinked) {
		t.Fatalf("err = %v, want ErrNotLinked", err)
	}
}

func TestRedeemSurfacesAStaleLinkToken(t *testing.T) {
	f := newFixture(t)
	f.auth.redeemErr = connect.NewError(connect.CodeUnauthenticated, errors.New("spent"))

	_, err := f.store.Redeem(context.Background(), "finance", "used", ada, 99)
	if !errors.Is(err, ErrLinkAgain) {
		t.Fatalf("err = %v, want ErrLinkAgain", err)
	}
}

func TestLinksArePerBot(t *testing.T) {
	f := newFixture(t)
	f.link(t)

	if _, err := f.store.Session(context.Background(), "notes", 42); !errors.Is(err, ErrNotLinked) {
		t.Fatalf("err = %v, want the notes bot to be unlinked", err)
	}
}

func TestUnlinkRevokesTheRefreshTokenAndDropsTheRow(t *testing.T) {
	f := newFixture(t)
	f.link(t)

	if err := f.store.Unlink(context.Background(), "finance", 42); err != nil {
		t.Fatalf("Unlink: %v", err)
	}
	if len(f.auth.loggedOut) != 1 || f.auth.loggedOut[0] != "refresh-0" {
		t.Fatalf("loggedOut = %v", f.auth.loggedOut)
	}
	if _, err := f.store.Session(context.Background(), "finance", 42); !errors.Is(err, ErrNotLinked) {
		t.Fatalf("err = %v, want ErrNotLinked after unlink", err)
	}
}

func TestUnlinkWithoutALink(t *testing.T) {
	f := newFixture(t)

	if err := f.store.Unlink(context.Background(), "finance", 42); !errors.Is(err, ErrNotLinked) {
		t.Fatalf("err = %v, want ErrNotLinked", err)
	}
}

func TestSessionReadsTheLocaleClaimFromTheToken(t *testing.T) {
	f := newFixture(t)
	f.auth.accessToken = jwtWithClaims(`{"sub":"` + testUserID + `","locale":"en"}`)
	f.link(t)

	s, err := f.store.Session(context.Background(), "finance", 42)
	if err != nil {
		t.Fatalf("Session: %v", err)
	}
	if s.Locale != "en" {
		t.Fatalf("locale = %q, want en", s.Locale)
	}
}

func TestSessionWithoutALocaleClaimIsBlank(t *testing.T) {
	f := newFixture(t)
	f.auth.accessToken = jwtWithClaims(`{"sub":"` + testUserID + `"}`)
	f.link(t)

	s, err := f.store.Session(context.Background(), "finance", 42)
	if err != nil {
		t.Fatalf("Session: %v", err)
	}
	if s.Locale != "" {
		t.Fatalf("locale = %q, want it empty so the caller falls back", s.Locale)
	}
}

func TestExpireAccessForcesTheNextCallToRefresh(t *testing.T) {
	f := newFixture(t)
	f.link(t)

	if err := f.store.ExpireAccess(context.Background(), "finance", 42); err != nil {
		t.Fatalf("ExpireAccess: %v", err)
	}

	if _, err := f.store.Session(context.Background(), "finance", 42); err != nil {
		t.Fatalf("Session: %v", err)
	}
	if len(f.auth.refreshed) != 1 {
		t.Fatalf("refreshed %d times, want the expiry to force exactly one", len(f.auth.refreshed))
	}
}
