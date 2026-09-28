package handler

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	fmauth "github.com/nnc/family-manager/libs/go/auth"
	authv1 "github.com/nnc/family-manager/sdk/go/auth/v1"
)

func asUser(userID string) context.Context {
	return fmauth.WithClaims(context.Background(), &fmauth.Claims{UserID: userID})
}

func (f *fixture) redeem(t *testing.T, tok, externalID string) (*authv1.RedeemLinkTokenResponse, error) {
	t.Helper()
	res, err := f.h.RedeemLinkToken(context.Background(),
		connect.NewRequest(&authv1.RedeemLinkTokenRequest{
			Token: tok, Provider: "telegram", ExternalId: externalID,
		}))
	if err != nil {
		return nil, err
	}
	return res.Msg, nil
}

func (f *fixture) refresh(refreshToken string) error {
	_, err := f.h.Refresh(context.Background(),
		connect.NewRequest(&authv1.RefreshRequest{RefreshToken: refreshToken}))
	return err
}

func (f *fixture) identities(t *testing.T, userID string) []*authv1.Identity {
	t.Helper()
	res, err := f.h.ListIdentities(asUser(userID), connect.NewRequest(&authv1.ListIdentitiesRequest{}))
	if err != nil {
		t.Fatalf("ListIdentities: %v", err)
	}
	return res.Msg.GetIdentities()
}

func TestRedeemBindsIdentity(t *testing.T) {
	f := newFixture(t)
	userID := f.register(t, "ada@example.test", "correct horse")

	if _, err := f.redeem(t, f.linkToken(t, userID), "4242"); err != nil {
		t.Fatalf("redeem: %v", err)
	}

	ids := f.identities(t, userID)
	if len(ids) != 1 || ids[0].GetProvider() != "telegram" || ids[0].GetExternalId() != "4242" {
		t.Fatalf("identities = %v, want one telegram:4242", ids)
	}
}

func TestRelinkRevokesPreviousSession(t *testing.T) {
	f := newFixture(t)
	userID := f.register(t, "ada@example.test", "correct horse")

	first, err := f.redeem(t, f.linkToken(t, userID), "4242")
	if err != nil {
		t.Fatalf("first redeem: %v", err)
	}
	second, err := f.redeem(t, f.linkToken(t, userID), "4242")
	if err != nil {
		t.Fatalf("second redeem: %v", err)
	}

	if connect.CodeOf(f.refresh(first.GetRefreshToken())) != connect.CodeUnauthenticated {
		t.Fatal("previous session still refreshes after relink")
	}
	if err := f.refresh(second.GetRefreshToken()); err != nil {
		t.Fatalf("new session refresh: %v", err)
	}
	if n := len(f.identities(t, userID)); n != 1 {
		t.Fatalf("identities = %d, want 1", n)
	}
}

func TestRedeemRejectsIdentityOwnedByAnotherUser(t *testing.T) {
	f := newFixture(t)
	ada := f.register(t, "ada@example.test", "correct horse")
	bob := f.register(t, "bob@example.test", "battery staple")

	if _, err := f.redeem(t, f.linkToken(t, ada), "4242"); err != nil {
		t.Fatalf("ada redeem: %v", err)
	}

	bobTok := f.linkToken(t, bob)
	_, err := f.redeem(t, bobTok, "4242")
	if connect.CodeOf(err) != connect.CodeAlreadyExists {
		t.Fatalf("code = %v, want already_exists", connect.CodeOf(err))
	}
	if ids := f.identities(t, bob); len(ids) != 0 {
		t.Fatalf("bob identities = %v, want none", ids)
	}

	if _, err := f.redeem(t, bobTok, "5555"); err != nil {
		t.Fatalf("rejected token was spent: %v", err)
	}
}

func TestUnlinkRevokesSession(t *testing.T) {
	f := newFixture(t)
	userID := f.register(t, "ada@example.test", "correct horse")

	res, err := f.redeem(t, f.linkToken(t, userID), "4242")
	if err != nil {
		t.Fatalf("redeem: %v", err)
	}

	if _, err := f.h.Unlink(asUser(userID), connect.NewRequest(&authv1.UnlinkRequest{
		Provider: "telegram", ExternalId: "4242",
	})); err != nil {
		t.Fatalf("Unlink: %v", err)
	}

	if connect.CodeOf(f.refresh(res.GetRefreshToken())) != connect.CodeUnauthenticated {
		t.Fatal("session still refreshes after unlink")
	}
	if ids := f.identities(t, userID); len(ids) != 0 {
		t.Fatalf("identities = %v, want none", ids)
	}
}

func TestUnlinkOnlyTouchesCallersIdentity(t *testing.T) {
	f := newFixture(t)
	ada := f.register(t, "ada@example.test", "correct horse")
	bob := f.register(t, "bob@example.test", "battery staple")

	if _, err := f.redeem(t, f.linkToken(t, ada), "4242"); err != nil {
		t.Fatalf("redeem: %v", err)
	}

	_, err := f.h.Unlink(asUser(bob), connect.NewRequest(&authv1.UnlinkRequest{
		Provider: "telegram", ExternalId: "4242",
	}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("code = %v, want not_found", connect.CodeOf(err))
	}
	if n := len(f.identities(t, ada)); n != 1 {
		t.Fatalf("ada identities = %d, want 1", n)
	}
}

func TestIdentityRPCsNeedClaims(t *testing.T) {
	f := newFixture(t)

	if _, err := f.h.ListIdentities(context.Background(),
		connect.NewRequest(&authv1.ListIdentitiesRequest{})); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("list code = %v, want unauthenticated", connect.CodeOf(err))
	}
	if _, err := f.h.Unlink(context.Background(), connect.NewRequest(&authv1.UnlinkRequest{
		Provider: "telegram", ExternalId: "4242",
	})); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("unlink code = %v, want unauthenticated", connect.CodeOf(err))
	}
}

func TestMintRefusesAChainRevokedBeforeItsFirstToken(t *testing.T) {
	f := newFixture(t)
	userID := f.register(t, "ada@example.test", "correct horse")

	chainID, err := newChainID()
	if err != nil {
		t.Fatalf("newChainID: %v", err)
	}
	if err := revokeChain(context.Background(), f.store, chainID); err != nil {
		t.Fatalf("revokeChain: %v", err)
	}

	_, err = f.h.mintSession(context.Background(), f.store.byID[userID], chainID, true)
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("code = %v, want unauthenticated", connect.CodeOf(err))
	}
}

func TestRefreshRefusesAChainWhoseRowIsGone(t *testing.T) {
	f := newFixture(t)
	userID := f.register(t, "ada@example.test", "correct horse")

	res, err := f.redeem(t, f.linkToken(t, userID), "4242")
	if err != nil {
		t.Fatalf("redeem: %v", err)
	}
	clear(f.store.chains)

	if connect.CodeOf(f.refresh(res.GetRefreshToken())) != connect.CodeUnauthenticated {
		t.Fatal("refresh recreated a swept chain")
	}
}

func (f *fixture) unlinkTelegram(t *testing.T, userID, externalID string) {
	t.Helper()
	if _, err := f.h.Unlink(asUser(userID), connect.NewRequest(&authv1.UnlinkRequest{
		Provider: "telegram", ExternalId: externalID,
	})); err != nil {
		t.Fatalf("Unlink: %v", err)
	}
}

func (f *fixture) signInApprovedBy(t *testing.T, access, userID string) *authv1.PollDeviceLoginResponse {
	t.Helper()
	start := f.startLogin(t, authv1.DeviceLoginKind_DEVICE_LOGIN_KIND_TELEGRAM)
	if err := f.approveAs(access, userID, start.GetUserCode()); err != nil {
		t.Fatalf("approve: %v", err)
	}
	res := f.poll(t, start.GetDeviceCode())
	if res.GetStatus() != authv1.DeviceLoginStatus_DEVICE_LOGIN_STATUS_APPROVED {
		t.Fatalf("poll = %v, want approved", res.GetStatus())
	}
	return res
}

func TestUnlinkRevokesSessionsTheIdentityApproved(t *testing.T) {
	f := newFixture(t)
	ada := f.register(t, "ada@example.test", "correct horse")
	tg, err := f.redeem(t, f.linkToken(t, ada), "4242")
	if err != nil {
		t.Fatalf("redeem: %v", err)
	}

	phone := f.signInApprovedBy(t, tg.GetAccessToken(), ada)
	tablet := f.signInApprovedBy(t, phone.GetAccessToken(), ada)
	ownAccess, _ := f.sessionOf(ada)
	laptop := f.signInApprovedBy(t, ownAccess, ada)

	f.unlinkTelegram(t, ada, "4242")

	if connect.CodeOf(f.refresh(phone.GetRefreshToken())) != connect.CodeUnauthenticated {
		t.Fatal("a session the Telegram identity approved survived unlink")
	}
	if connect.CodeOf(f.refresh(tablet.GetRefreshToken())) != connect.CodeUnauthenticated {
		t.Fatal("a session approved from a Telegram-approved session survived unlink")
	}
	if err := f.refresh(laptop.GetRefreshToken()); err != nil {
		t.Fatalf("a session approved elsewhere was revoked: %v", err)
	}
}

func TestUnlinkStopsAnApprovedGrantFromMinting(t *testing.T) {
	f := newFixture(t)
	ada := f.register(t, "ada@example.test", "correct horse")
	tg, err := f.redeem(t, f.linkToken(t, ada), "4242")
	if err != nil {
		t.Fatalf("redeem: %v", err)
	}
	start := f.startLogin(t, authv1.DeviceLoginKind_DEVICE_LOGIN_KIND_TELEGRAM)
	if err := f.approveAs(tg.GetAccessToken(), ada, start.GetUserCode()); err != nil {
		t.Fatalf("approve: %v", err)
	}

	f.unlinkTelegram(t, ada, "4242")

	res := f.poll(t, start.GetDeviceCode())
	if res.GetStatus() != authv1.DeviceLoginStatus_DEVICE_LOGIN_STATUS_EXPIRED || res.GetAccessToken() != "" {
		t.Fatalf("poll after unlink = %v with a token %t, want expired and empty",
			res.GetStatus(), res.GetAccessToken() != "")
	}
}

func TestRelinkRevokesSessionsThePreviousLinkApproved(t *testing.T) {
	f := newFixture(t)
	ada := f.register(t, "ada@example.test", "correct horse")
	first, err := f.redeem(t, f.linkToken(t, ada), "4242")
	if err != nil {
		t.Fatalf("first redeem: %v", err)
	}
	phone := f.signInApprovedBy(t, first.GetAccessToken(), ada)

	if _, err := f.redeem(t, f.linkToken(t, ada), "4242"); err != nil {
		t.Fatalf("second redeem: %v", err)
	}

	if connect.CodeOf(f.refresh(phone.GetRefreshToken())) != connect.CodeUnauthenticated {
		t.Fatal("a session the replaced link approved survived relink")
	}
}
