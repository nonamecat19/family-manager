package handler

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"

	fmauth "github.com/nnc/family-manager/libs/go/auth"
	"github.com/nnc/family-manager/libs/go/database/pgconv"
	notificationsv1 "github.com/nnc/family-manager/sdk/go/notifications/v1"
	"github.com/nnc/family-manager/services/notifications/internal/dbtest"
	"github.com/nnc/family-manager/services/notifications/internal/topics"
)

const (
	alice  = "11111111-1111-4111-8111-111111111111"
	bob    = "22222222-2222-4222-8222-222222222222"
	family = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"

	tokenA = "ExponentPushToken[aaaaaaaaaaaaaaaaaaaaaa]"
)

func newHandler() (*Handler, *dbtest.Fake) {
	store := dbtest.New()
	return New(Options{Queries: store, Tx: store}), store
}

func asUser(userID, familyID string) context.Context {
	return fmauth.WithClaims(context.Background(), &fmauth.Claims{UserID: userID, FamilyID: familyID})
}

func register(token string, app notificationsv1.App) *connect.Request[notificationsv1.RegisterPushTokenRequest] {
	return connect.NewRequest(&notificationsv1.RegisterPushTokenRequest{
		Token:    token,
		Platform: notificationsv1.Platform_PLATFORM_ANDROID,
		App:      app,
		DeviceId: "pixel-8",
	})
}

func codeOf(err error) connect.Code {
	var ce *connect.Error
	if errors.As(err, &ce) {
		return ce.Code()
	}
	return connect.Code(0)
}

func TestRegisterStoresTheTokenAgainstTheCallerAndFamily(t *testing.T) {
	h, store := newHandler()

	if _, err := h.RegisterPushToken(asUser(alice, family), register(tokenA, notificationsv1.App_APP_FINANCE)); err != nil {
		t.Fatalf("RegisterPushToken: %v", err)
	}

	got, ok := store.Tokens[tokenA]
	if !ok {
		t.Fatal("token was not stored")
	}
	if pgconv.UUIDString(got.UserID) != alice || pgconv.UUIDString(got.FamilyID) != family {
		t.Errorf("owner = %s family = %s", pgconv.UUIDString(got.UserID), pgconv.UUIDString(got.FamilyID))
	}
	if got.App != "finance" || got.Platform != "android" || got.DeviceID != "pixel-8" {
		t.Errorf("stored %+v", got)
	}
}

func TestRegisterMovesATokenToItsNewOwner(t *testing.T) {
	h, store := newHandler()
	_, _ = h.RegisterPushToken(asUser(alice, family), register(tokenA, notificationsv1.App_APP_NOTES))
	if _, err := h.RegisterPushToken(asUser(bob, ""), register(tokenA, notificationsv1.App_APP_NOTES)); err != nil {
		t.Fatalf("RegisterPushToken: %v", err)
	}
	got := store.Tokens[tokenA]
	if pgconv.UUIDString(got.UserID) != bob {
		t.Errorf("owner = %s, want bob", pgconv.UUIDString(got.UserID))
	}
	if got.FamilyID.Valid {
		t.Error("the previous owner's family stuck to the token")
	}
}

func TestRegisterDoesNotMoveATokenFromAnotherDevice(t *testing.T) {
	h, store := newHandler()
	_, _ = h.RegisterPushToken(asUser(alice, family), register(tokenA, notificationsv1.App_APP_NOTES))

	stolen := register(tokenA, notificationsv1.App_APP_NOTES)
	stolen.Msg.DeviceId = "attacker-phone"
	_, err := h.RegisterPushToken(asUser(bob, ""), stolen)
	if codeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("err = %v, want permission denied", err)
	}

	blank := register(tokenA, notificationsv1.App_APP_NOTES)
	blank.Msg.DeviceId = ""
	if _, err := h.RegisterPushToken(asUser(bob, ""), blank); codeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("without a device id: err = %v, want permission denied", err)
	}

	got := store.Tokens[tokenA]
	if pgconv.UUIDString(got.UserID) != alice || pgconv.UUIDString(got.FamilyID) != family {
		t.Errorf("token rebound to %s / %s", pgconv.UUIDString(got.UserID), pgconv.UUIDString(got.FamilyID))
	}
}

func TestRegisterLetsTheOwnerUpdateTheirToken(t *testing.T) {
	h, store := newHandler()
	_, _ = h.RegisterPushToken(asUser(alice, family), register(tokenA, notificationsv1.App_APP_NOTES))

	again := register(tokenA, notificationsv1.App_APP_FINANCE)
	again.Msg.DeviceId = "reinstalled"
	if _, err := h.RegisterPushToken(asUser(alice, family), again); err != nil {
		t.Fatalf("RegisterPushToken: %v", err)
	}
	if got := store.Tokens[tokenA]; got.App != "finance" || got.DeviceID != "reinstalled" {
		t.Errorf("stored %+v", got)
	}
}

func TestRegisterValidates(t *testing.T) {
	h, store := newHandler()
	cases := map[string]*notificationsv1.RegisterPushTokenRequest{
		"not expo":    {Token: "fcm:abc", Platform: notificationsv1.Platform_PLATFORM_IOS, App: notificationsv1.App_APP_NOTES},
		"no platform": {Token: tokenA, App: notificationsv1.App_APP_NOTES},
		"no app":      {Token: tokenA, Platform: notificationsv1.Platform_PLATFORM_IOS},
		"long device": {
			Token: tokenA, Platform: notificationsv1.Platform_PLATFORM_IOS, App: notificationsv1.App_APP_NOTES,
			DeviceId: string(make([]rune, maxDeviceRunes+1)),
		},
	}
	for name, msg := range cases {
		_, err := h.RegisterPushToken(asUser(alice, family), connect.NewRequest(msg))
		if codeOf(err) != connect.CodeInvalidArgument {
			t.Errorf("%s: err = %v, want invalid argument", name, err)
		}
	}
	if len(store.Tokens) != 0 {
		t.Errorf("stored %d invalid tokens", len(store.Tokens))
	}
}

func TestRegisterAcceptsBothExpoTokenSpellings(t *testing.T) {
	for _, tok := range []string{"ExponentPushToken[x]", "ExpoPushToken[x]"} {
		if !ValidToken(tok) {
			t.Errorf("ValidToken(%q) = false", tok)
		}
	}
	for _, tok := range []string{"", "ExponentPushToken[x", "ExpoPushToken"} {
		if ValidToken(tok) {
			t.Errorf("ValidToken(%q) = true", tok)
		}
	}
}

func TestEveryRPCRequiresAUser(t *testing.T) {
	h, _ := newHandler()
	ctx := context.Background()

	_, err1 := h.RegisterPushToken(ctx, register(tokenA, notificationsv1.App_APP_NOTES))
	_, err2 := h.UnregisterPushToken(ctx, connect.NewRequest(&notificationsv1.UnregisterPushTokenRequest{Token: tokenA}))
	_, err3 := h.GetPreferences(ctx, connect.NewRequest(&notificationsv1.GetPreferencesRequest{}))
	_, err4 := h.SetPreferences(ctx, connect.NewRequest(&notificationsv1.SetPreferencesRequest{}))
	for i, err := range []error{err1, err2, err3, err4} {
		if codeOf(err) != connect.CodeUnauthenticated {
			t.Errorf("rpc %d: err = %v, want unauthenticated", i, err)
		}
	}
}

func TestUnregisterOnlyRemovesTheCallersToken(t *testing.T) {
	h, store := newHandler()
	_, _ = h.RegisterPushToken(asUser(alice, family), register(tokenA, notificationsv1.App_APP_NOTES))

	req := connect.NewRequest(&notificationsv1.UnregisterPushTokenRequest{Token: tokenA})
	if _, err := h.UnregisterPushToken(asUser(bob, family), req); err != nil {
		t.Fatalf("UnregisterPushToken as bob: %v", err)
	}
	if _, ok := store.Tokens[tokenA]; !ok {
		t.Fatal("bob removed alice's token")
	}
	if _, err := h.UnregisterPushToken(asUser(alice, family), req); err != nil {
		t.Fatalf("UnregisterPushToken as alice: %v", err)
	}
	if _, ok := store.Tokens[tokenA]; ok {
		t.Fatal("alice's token survived her unregister")
	}
}

func TestPreferencesRoundTrip(t *testing.T) {
	h, _ := newHandler()
	ctx := asUser(alice, family)

	set, err := h.SetPreferences(ctx, connect.NewRequest(&notificationsv1.SetPreferencesRequest{
		Muted: []string{topics.RecipesRecipeCreated, "finance", " finance "},
	}))
	if err != nil {
		t.Fatalf("SetPreferences: %v", err)
	}
	want := []string{"finance", topics.RecipesRecipeCreated}
	if got := set.Msg.GetMuted(); len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("muted = %v, want %v", got, want)
	}

	got, err := h.GetPreferences(ctx, connect.NewRequest(&notificationsv1.GetPreferencesRequest{}))
	if err != nil {
		t.Fatalf("GetPreferences: %v", err)
	}
	if m := got.Msg.GetMuted(); len(m) != 2 || m[0] != want[0] || m[1] != want[1] {
		t.Errorf("muted = %v, want %v", m, want)
	}
	if len(got.Msg.GetTopics()) != len(topics.All) {
		t.Errorf("topics = %d, want %d", len(got.Msg.GetTopics()), len(topics.All))
	}

	cleared, err := h.SetPreferences(ctx, connect.NewRequest(&notificationsv1.SetPreferencesRequest{}))
	if err != nil {
		t.Fatalf("SetPreferences(empty): %v", err)
	}
	if len(cleared.Msg.GetMuted()) != 0 {
		t.Errorf("muted after clear = %v", cleared.Msg.GetMuted())
	}
}

func TestSetPreferencesRejectsUnknownTopics(t *testing.T) {
	h, store := newHandler()
	_, err := h.SetPreferences(asUser(alice, family), connect.NewRequest(&notificationsv1.SetPreferencesRequest{
		Muted: []string{"finance", "shopping.list.completed"},
	}))
	if codeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("err = %v, want invalid argument", err)
	}
	if len(store.Mutes) != 0 {
		t.Error("a rejected request still wrote mutes")
	}
}

func TestStoreFailuresAreInternal(t *testing.T) {
	h, store := newHandler()
	store.FailOn["ListMutes"] = errors.New("connection reset")
	_, err := h.GetPreferences(asUser(alice, family), connect.NewRequest(&notificationsv1.GetPreferencesRequest{}))
	if codeOf(err) != connect.CodeInternal {
		t.Fatalf("err = %v, want internal", err)
	}
}
