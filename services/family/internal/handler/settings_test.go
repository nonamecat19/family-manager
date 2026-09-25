package handler

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	fmauth "github.com/nnc/family-manager/libs/go/auth"
	familyv1 "github.com/nnc/family-manager/sdk/go/family/v1"
)

const settingsUser = "6f1b9d3e-0d3a-4c19-9a2f-2f9a6a3d5c11"

func callerCtx(userID string) context.Context {
	return fmauth.WithClaims(context.Background(), &fmauth.Claims{UserID: userID})
}

func TestMySettingsDefaultsToUkrainian(t *testing.T) {
	f := newFixture(t)

	res, err := f.h.GetMySettings(callerCtx(settingsUser),
		connect.NewRequest(&familyv1.GetMySettingsRequest{}))
	if err != nil {
		t.Fatalf("GetMySettings: %v", err)
	}
	if got := res.Msg.GetSettings().GetLocale(); got != DefaultLocale {
		t.Fatalf("locale = %q, want %q", got, DefaultLocale)
	}
}

func TestSettingsWorkWithoutAFamily(t *testing.T) {
	f := newFixture(t)

	if _, err := f.h.UpdateMySettings(callerCtx(settingsUser),
		connect.NewRequest(&familyv1.UpdateMySettingsRequest{Locale: "en"})); err != nil {
		t.Fatalf("UpdateMySettings for a user in no family: %v", err)
	}

	res, err := f.h.GetMySettings(callerCtx(settingsUser),
		connect.NewRequest(&familyv1.GetMySettingsRequest{}))
	if err != nil {
		t.Fatalf("GetMySettings: %v", err)
	}
	if got := res.Msg.GetSettings().GetLocale(); got != "en" {
		t.Fatalf("locale = %q, want en", got)
	}
}

func TestUpdateRejectsAnUnsupportedLocale(t *testing.T) {
	f := newFixture(t)

	_, err := f.h.UpdateMySettings(callerCtx(settingsUser),
		connect.NewRequest(&familyv1.UpdateMySettingsRequest{Locale: "fr"}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want invalid_argument", connect.CodeOf(err))
	}
}

func TestUpdateKeepsUntouchedFields(t *testing.T) {
	f := newFixture(t)
	ctx := callerCtx(settingsUser)

	if _, err := f.h.UpdateMySettings(ctx, connect.NewRequest(&familyv1.UpdateMySettingsRequest{
		Locale: "en", Timezone: "Europe/Kyiv",
	})); err != nil {
		t.Fatalf("UpdateMySettings: %v", err)
	}

	res, err := f.h.UpdateMySettings(ctx,
		connect.NewRequest(&familyv1.UpdateMySettingsRequest{Locale: "uk"}))
	if err != nil {
		t.Fatalf("UpdateMySettings: %v", err)
	}
	if got := res.Msg.GetSettings().GetTimezone(); got != "Europe/Kyiv" {
		t.Fatalf("timezone = %q, want it preserved", got)
	}
	if got := res.Msg.GetSettings().GetLocale(); got != "uk" {
		t.Fatalf("locale = %q, want uk", got)
	}
}

func TestSettingsNeedAuthentication(t *testing.T) {
	f := newFixture(t)

	_, err := f.h.GetMySettings(context.Background(),
		connect.NewRequest(&familyv1.GetMySettingsRequest{}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("code = %v, want unauthenticated", connect.CodeOf(err))
	}
}

func TestGetUserSettingsIsLookedUpByID(t *testing.T) {
	f := newFixture(t)

	if _, err := f.h.UpdateMySettings(callerCtx(settingsUser),
		connect.NewRequest(&familyv1.UpdateMySettingsRequest{Locale: "en"})); err != nil {
		t.Fatalf("UpdateMySettings: %v", err)
	}

	res, err := f.h.GetUserSettings(context.Background(),
		connect.NewRequest(&familyv1.GetUserSettingsRequest{UserId: settingsUser}))
	if err != nil {
		t.Fatalf("GetUserSettings: %v", err)
	}
	if got := res.Msg.GetSettings().GetLocale(); got != "en" {
		t.Fatalf("locale = %q, want en", got)
	}

	if _, err := f.h.GetUserSettings(context.Background(),
		connect.NewRequest(&familyv1.GetUserSettingsRequest{})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want invalid_argument for a missing user id", connect.CodeOf(err))
	}
}
