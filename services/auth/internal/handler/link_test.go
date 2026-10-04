package handler

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"

	fmauth "github.com/nnc/family-manager/libs/go/auth"
	authv1 "github.com/nnc/family-manager/sdk/go/auth/v1"
)

func (f *fixture) linkToken(t *testing.T, userID string) string {
	t.Helper()
	access, _ := f.sessionOf(userID)
	tok, err := f.linkTokenAs(access, userID)
	if err != nil {
		t.Fatalf("CreateLinkToken: %v", err)
	}
	return tok
}

func (f *fixture) linkTokenAs(access, userID string) (string, error) {
	ctx := fmauth.WithClaims(context.Background(), &fmauth.Claims{UserID: userID})
	req := connect.NewRequest(&authv1.CreateLinkTokenRequest{Provider: "telegram"})
	req.Header().Set("Authorization", "Bearer "+access)
	res, err := f.h.CreateLinkToken(ctx, req)
	if err != nil {
		return "", err
	}
	return res.Msg.GetToken(), nil
}

func TestCreateLinkTokenNeedsClaims(t *testing.T) {
	f := newFixture(t)

	_, err := f.h.CreateLinkToken(context.Background(),
		connect.NewRequest(&authv1.CreateLinkTokenRequest{Provider: "telegram"}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("code = %v, want unauthenticated", connect.CodeOf(err))
	}
}

func TestCreateLinkTokenRejectsUnknownProvider(t *testing.T) {
	f := newFixture(t)
	userID := f.register(t, "ada@example.test", "correct horse")

	ctx := fmauth.WithClaims(context.Background(), &fmauth.Claims{UserID: userID})
	_, err := f.h.CreateLinkToken(ctx,
		connect.NewRequest(&authv1.CreateLinkTokenRequest{Provider: "signal"}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want invalid_argument", connect.CodeOf(err))
	}
}

func TestRedeemLinkTokenMintsASession(t *testing.T) {
	f := newFixture(t)
	userID := f.register(t, "ada@example.test", "correct horse")
	tok := f.linkToken(t, userID)

	res, err := f.h.RedeemLinkToken(context.Background(),
		connect.NewRequest(&authv1.RedeemLinkTokenRequest{
			Token: tok, Provider: "telegram", ExternalId: "4242",
		}))
	if err != nil {
		t.Fatalf("RedeemLinkToken: %v", err)
	}

	if res.Msg.GetUserId() != userID {
		t.Fatalf("user id = %q, want %q", res.Msg.GetUserId(), userID)
	}
	if res.Msg.GetAccessToken() == "" || res.Msg.GetRefreshToken() == "" {
		t.Fatal("redeem returned an empty token pair")
	}
}

func TestRedeemLinkTokenIsSingleUse(t *testing.T) {
	f := newFixture(t)
	userID := f.register(t, "ada@example.test", "correct horse")
	tok := f.linkToken(t, userID)

	req := func() *connect.Request[authv1.RedeemLinkTokenRequest] {
		return connect.NewRequest(&authv1.RedeemLinkTokenRequest{
			Token: tok, Provider: "telegram", ExternalId: "4242",
		})
	}

	if _, err := f.h.RedeemLinkToken(context.Background(), req()); err != nil {
		t.Fatalf("first redeem: %v", err)
	}

	_, err := f.h.RedeemLinkToken(context.Background(), req())
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("code = %v, want unauthenticated", connect.CodeOf(err))
	}
}

func TestRedeemLinkTokenRejectsAnotherProvider(t *testing.T) {
	f := newFixture(t)
	userID := f.register(t, "ada@example.test", "correct horse")
	tok := f.linkToken(t, userID)

	_, err := f.h.RedeemLinkToken(context.Background(),
		connect.NewRequest(&authv1.RedeemLinkTokenRequest{
			Token: tok, Provider: "whatsapp", ExternalId: "4242",
		}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want invalid_argument", connect.CodeOf(err))
	}
}

func TestRedeemLinkTokenRejectsAnExpiredToken(t *testing.T) {
	f := newFixture(t)
	userID := f.register(t, "ada@example.test", "correct horse")
	tok := f.linkToken(t, userID)

	f.h.now = func() time.Time { return time.Now().Add(linkTokenTTL + time.Minute) }

	_, err := f.h.RedeemLinkToken(context.Background(),
		connect.NewRequest(&authv1.RedeemLinkTokenRequest{
			Token: tok, Provider: "telegram", ExternalId: "4242",
		}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("code = %v, want unauthenticated", connect.CodeOf(err))
	}
}

func TestRedeemLinkTokenRequiresAnExternalID(t *testing.T) {
	f := newFixture(t)
	userID := f.register(t, "ada@example.test", "correct horse")
	tok := f.linkToken(t, userID)

	_, err := f.h.RedeemLinkToken(context.Background(),
		connect.NewRequest(&authv1.RedeemLinkTokenRequest{Token: tok, Provider: "telegram"}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want invalid_argument", connect.CodeOf(err))
	}
}

func TestSessionsCarryTheUsersLocale(t *testing.T) {
	f := newFixture(t)
	f.family.locale = "en"
	f.register(t, "ada@example.test", "correct horse")

	f.login(t, "ada@example.test", "correct horse")

	if got := f.signer.last().Locale; got != "en" {
		t.Fatalf("locale claim = %q, want en", got)
	}
}

func TestLocaleLookupFailureStillMintsAToken(t *testing.T) {
	f := newFixture(t)
	f.register(t, "ada@example.test", "correct horse")
	f.family.err = errBoom

	res := f.login(t, "ada@example.test", "correct horse")

	if res.GetAccessToken() == "" {
		t.Fatal("a settings outage blocked sign-in")
	}
	if got := f.signer.last().Locale; got != "" {
		t.Fatalf("locale claim = %q, want it omitted", got)
	}
}

func TestCreateLinkTokenRefusesIdentityDerivedSessions(t *testing.T) {
	f := newFixture(t)
	ada := f.register(t, "ada@example.test", "correct horse")
	tg, err := f.redeem(t, f.linkToken(t, ada), "4242")
	if err != nil {
		t.Fatalf("redeem: %v", err)
	}
	phone := f.signInApprovedBy(t, tg.GetAccessToken(), ada)
	tablet := f.signInApprovedBy(t, phone.GetAccessToken(), ada)

	for name, access := range map[string]string{
		"the identity's own session":   tg.GetAccessToken(),
		"a session it approved":        phone.GetAccessToken(),
		"a session approved from that": tablet.GetAccessToken(),
	} {
		if _, err := f.linkTokenAs(access, ada); connect.CodeOf(err) != connect.CodePermissionDenied {
			t.Errorf("%s: CreateLinkToken = %v, want permission_denied", name, connect.CodeOf(err))
		}
	}
}

func TestCreateLinkTokenAllowsSessionsRootedAtAPasswordLogin(t *testing.T) {
	f := newFixture(t)
	ada := f.register(t, "ada@example.test", "correct horse")
	password, _ := f.sessionOf(ada)
	laptop := f.signInApprovedBy(t, password, ada)

	if _, err := f.linkTokenAs(password, ada); err != nil {
		t.Fatalf("password session: %v", err)
	}
	if _, err := f.linkTokenAs(laptop.GetAccessToken(), ada); err != nil {
		t.Fatalf("device session approved from a password session: %v", err)
	}
}

func TestCreateLinkTokenRefusesARevokedSession(t *testing.T) {
	f := newFixture(t)
	ada := f.register(t, "ada@example.test", "correct horse")
	access, chainID := f.sessionOf(ada)
	if err := revokeChain(context.Background(), f.store, chainID); err != nil {
		t.Fatalf("revoke: %v", err)
	}

	if _, err := f.linkTokenAs(access, ada); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("CreateLinkToken from a revoked session = %v, want unauthenticated", connect.CodeOf(err))
	}
	if len(f.store.links) != 0 {
		t.Fatal("a revoked session stored a link token")
	}
}

func TestUnlinkedIdentityCannotOutliveItsUnlinkThroughARelink(t *testing.T) {
	f := newFixture(t)
	ada := f.register(t, "ada@example.test", "correct horse")
	tg, err := f.redeem(t, f.linkToken(t, ada), "4242")
	if err != nil {
		t.Fatalf("redeem: %v", err)
	}
	f.unlinkTelegram(t, ada, "4242")

	if _, err := f.linkTokenAs(tg.GetAccessToken(), ada); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("CreateLinkToken from an unlinked identity's live access token = %v, want unauthenticated",
			connect.CodeOf(err))
	}
}
