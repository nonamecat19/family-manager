package session

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/nnc/family-manager/libs/go/database/pgconv"
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
	s, err := f.store.Redeem(context.Background(), "link-token", ada, 99)
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

	link, err := f.q.GetLink(context.Background(), 42)
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

	s, err := f.store.Session(context.Background(), 42)
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

	s, err := f.store.Session(context.Background(), 42)
	if err != nil {
		t.Fatalf("Session: %v", err)
	}
	if s.AccessToken != "access-1" {
		t.Fatalf("access token = %q, want a refreshed one", s.AccessToken)
	}
	if len(f.auth.refreshed) != 1 || f.auth.refreshed[0] != "refresh-0" {
		t.Fatalf("refreshed = %v", f.auth.refreshed)
	}

	again, err := f.store.Session(context.Background(), 42)
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

	_, err := f.store.Session(context.Background(), 42)
	if !errors.Is(err, ErrLinkAgain) {
		t.Fatalf("err = %v, want ErrLinkAgain", err)
	}
	if len(f.q.deleted) != 1 {
		t.Fatal("the dead link was kept")
	}
}

func TestSessionWithoutALink(t *testing.T) {
	f := newFixture(t)

	_, err := f.store.Session(context.Background(), 42)
	if !errors.Is(err, ErrNotLinked) {
		t.Fatalf("err = %v, want ErrNotLinked", err)
	}
}

func TestRedeemSurfacesAStaleLinkToken(t *testing.T) {
	f := newFixture(t)
	f.auth.redeemErr = connect.NewError(connect.CodeUnauthenticated, errors.New("spent"))

	_, err := f.store.Redeem(context.Background(), "used", ada, 99)
	if !errors.Is(err, ErrLinkAgain) {
		t.Fatalf("err = %v, want ErrLinkAgain", err)
	}
}

func TestRedeemReportsAnAccountLinkedElsewhere(t *testing.T) {
	f := newFixture(t)
	f.auth.redeemErr = connect.NewError(connect.CodeAlreadyExists, errors.New("taken"))

	if _, err := f.store.Redeem(context.Background(), "link-token", ada, 99); !errors.Is(err, ErrTaken) {
		t.Fatalf("err = %v, want ErrTaken", err)
	}
	if len(f.q.links) != 0 {
		t.Fatalf("links = %v, want none stored", f.q.links)
	}
}

func TestUnlinkRemovesTheIdentityInAuthAndDropsTheRow(t *testing.T) {
	f := newFixture(t)
	f.link(t)

	if err := f.store.Unlink(context.Background(), 42); err != nil {
		t.Fatalf("Unlink: %v", err)
	}
	if len(f.auth.unlinked) != 1 ||
		f.auth.unlinked[0].GetProvider() != "telegram" || f.auth.unlinked[0].GetExternalId() != "42" {
		t.Fatalf("unlinked = %v, want telegram:42", f.auth.unlinked)
	}
	if f.auth.bearers[0] != "Bearer access-0" {
		t.Fatalf("bearer = %q, want the user's access token", f.auth.bearers[0])
	}
	if _, err := f.store.Session(context.Background(), 42); !errors.Is(err, ErrNotLinked) {
		t.Fatalf("err = %v, want ErrNotLinked after unlink", err)
	}
}

func TestUnlinkAlreadyGoneInAuthStillDropsTheRow(t *testing.T) {
	f := newFixture(t)
	f.link(t)
	f.auth.unlinkErr = connect.NewError(connect.CodeNotFound, errors.New("gone"))

	if err := f.store.Unlink(context.Background(), 42); err != nil {
		t.Fatalf("Unlink: %v", err)
	}
	if _, err := f.store.Session(context.Background(), 42); !errors.Is(err, ErrNotLinked) {
		t.Fatalf("err = %v, want ErrNotLinked after unlink", err)
	}
}

func TestUnlinkAfterAuthRevokedTheSession(t *testing.T) {
	f := newFixture(t)
	f.link(t)
	f.now = f.now.Add(time.Hour)
	f.auth.refreshErr = connect.NewError(connect.CodeUnauthenticated, errors.New("revoked"))

	if err := f.store.Unlink(context.Background(), 42); err != nil {
		t.Fatalf("Unlink: %v", err)
	}
	if len(f.auth.unlinked) != 0 {
		t.Fatalf("unlinked = %v, want no call with a dead session", f.auth.unlinked)
	}
	if len(f.q.links) != 0 {
		t.Fatalf("links = %v, want the dead row dropped", f.q.links)
	}
}

func TestUnlinkWithoutALink(t *testing.T) {
	f := newFixture(t)

	if err := f.store.Unlink(context.Background(), 42); !errors.Is(err, ErrNotLinked) {
		t.Fatalf("err = %v, want ErrNotLinked", err)
	}
}

func TestSessionReadsTheLocaleClaimFromTheToken(t *testing.T) {
	f := newFixture(t)
	f.auth.accessToken = jwtWithClaims(`{"sub":"` + testUserID + `","locale":"en"}`)
	f.link(t)

	s, err := f.store.Session(context.Background(), 42)
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

	s, err := f.store.Session(context.Background(), 42)
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

	if err := f.store.ExpireAccess(context.Background(), 42); err != nil {
		t.Fatalf("ExpireAccess: %v", err)
	}

	if _, err := f.store.Session(context.Background(), 42); err != nil {
		t.Fatalf("Session: %v", err)
	}
	if len(f.auth.refreshed) != 1 {
		t.Fatalf("refreshed %d times, want the expiry to force exactly one", len(f.auth.refreshed))
	}
}

const otherUserID = "0c7e2a51-8d44-4f0b-b3a6-5b1e9f2d7a90"

func (f *fixture) relinkToOtherUser(t *testing.T) {
	t.Helper()
	if _, err := f.q.UpsertLink(context.Background(), db.UpsertLinkParams{
		TelegramUserID:  42,
		UserID:          pgconv.MustUUID(otherUserID),
		ChatID:          99,
		AccessToken:     []byte("bob-access"),
		AccessExpiresAt: pgconv.TimestampFrom(f.now.Add(time.Hour)),
		RefreshToken:    []byte("bob-refresh"),
	}); err != nil {
		t.Fatalf("UpsertLink: %v", err)
	}
}

func TestConcurrentSessionsRefreshOnce(t *testing.T) {
	f := newFixture(t)
	f.link(t)
	f.now = f.now.Add(time.Hour)

	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := f.store.Session(context.Background(), 42)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("Session: %v", err)
		}
	}
	if len(f.auth.refreshed) != 1 {
		t.Fatalf("refreshed %d times, want once", len(f.auth.refreshed))
	}
}

func TestStaleRefreshDoesNotOverwriteANewerLink(t *testing.T) {
	f := newFixture(t)
	f.link(t)
	f.now = f.now.Add(time.Hour)
	f.auth.onRefresh = func() { f.relinkToOtherUser(t) }

	if _, err := f.store.Session(context.Background(), 42); !errors.Is(err, ErrLinkAgain) {
		t.Fatalf("err = %v, want ErrLinkAgain", err)
	}

	link, err := f.q.GetLink(context.Background(), 42)
	if err != nil {
		t.Fatalf("GetLink: %v", err)
	}
	if pgconv.UUIDString(link.UserID) != otherUserID || string(link.RefreshToken) != "bob-refresh" {
		t.Fatalf("link = %+v, want the newer link untouched", link)
	}
}

func TestDeadRefreshKeepsANewerLink(t *testing.T) {
	f := newFixture(t)
	f.link(t)
	f.now = f.now.Add(time.Hour)
	f.auth.refreshErr = connect.NewError(connect.CodeUnauthenticated, errors.New("revoked"))
	f.auth.onRefresh = func() { f.relinkToOtherUser(t) }

	if _, err := f.store.Session(context.Background(), 42); !errors.Is(err, ErrLinkAgain) {
		t.Fatalf("err = %v, want ErrLinkAgain", err)
	}
	if _, err := f.q.GetLink(context.Background(), 42); err != nil {
		t.Fatalf("newer link was dropped: %v", err)
	}
}

func TestApproveLoginActsAsTheLinkedUser(t *testing.T) {
	f := newFixture(t)
	f.link(t)

	if err := f.store.ApproveLogin(context.Background(), 42, "BCDFGHJK"); err != nil {
		t.Fatalf("ApproveLogin: %v", err)
	}
	if len(f.auth.approved) != 1 || f.auth.approved[0].GetUserCode() != "BCDFGHJK" {
		t.Fatalf("approved = %v, want the code passed through", f.auth.approved)
	}
	if f.auth.bearers[0] != "Bearer access-0" {
		t.Fatalf("bearer = %q, want the linked user's access token", f.auth.bearers[0])
	}
}

func TestDenyLoginActsAsTheLinkedUser(t *testing.T) {
	f := newFixture(t)
	f.link(t)

	if err := f.store.DenyLogin(context.Background(), 42, "BCDFGHJK"); err != nil {
		t.Fatalf("DenyLogin: %v", err)
	}
	if len(f.auth.denied) != 1 || f.auth.bearers[0] != "Bearer access-0" {
		t.Fatalf("denied = %v bearers = %v", f.auth.denied, f.auth.bearers)
	}
}

func TestApproveLoginWithoutALinkNeverReachesAuth(t *testing.T) {
	f := newFixture(t)

	if err := f.store.ApproveLogin(context.Background(), 42, "BCDFGHJK"); !errors.Is(err, ErrNotLinked) {
		t.Fatalf("err = %v, want ErrNotLinked", err)
	}
	if len(f.auth.bearers) != 0 {
		t.Fatal("an unlinked telegram user reached ApproveDeviceLogin")
	}
}

func TestApproveLoginMapsAuthRefusals(t *testing.T) {
	cases := map[connect.Code]error{
		connect.CodeNotFound:          ErrLoginExpired,
		connect.CodeInvalidArgument:   ErrLoginExpired,
		connect.CodeResourceExhausted: ErrLoginThrottled,
	}
	for code, want := range cases {
		f := newFixture(t)
		f.link(t)
		f.auth.decideErr = connect.NewError(code, errors.New("refused"))

		if err := f.store.ApproveLogin(context.Background(), 42, "BCDFGHJK"); !errors.Is(err, want) {
			t.Errorf("%v: err = %v, want %v", code, err, want)
		}
	}

	f := newFixture(t)
	f.link(t)
	f.auth.decideErr = connect.NewError(connect.CodeUnavailable, errors.New("down"))
	err := f.store.ApproveLogin(context.Background(), 42, "BCDFGHJK")
	if err == nil || errors.Is(err, ErrLoginExpired) || errors.Is(err, ErrLoginThrottled) {
		t.Fatalf("err = %v, want an opaque failure", err)
	}
}
