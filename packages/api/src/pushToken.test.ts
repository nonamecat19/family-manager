import assert from "node:assert/strict";
import { test } from "node:test";

import type { App as WireApp, Platform as WirePlatform } from "@fm/sdk/notifications/v1/notifications_pb";

import { appToProto, isExpoPushToken, platformToProto } from "./pushToken.ts";

const App: Record<"NOTES" | "FINANCE" | "RECIPES", WireApp> = { NOTES: 1, FINANCE: 2, RECIPES: 3 };
const Platform: Record<"UNSPECIFIED" | "IOS" | "ANDROID", WirePlatform> = {
  UNSPECIFIED: 0,
  IOS: 1,
  ANDROID: 2,
};

test("app names map to the proto enum the service expects", () => {
  assert.equal(appToProto("notes"), App.NOTES);
  assert.equal(appToProto("finance"), App.FINANCE);
  assert.equal(appToProto("recipes"), App.RECIPES);
});

test("react-native's Platform.OS maps to the proto enum, unspecified off ios/android", () => {
  assert.equal(platformToProto("ios"), Platform.IOS);
  assert.equal(platformToProto("android"), Platform.ANDROID);
  assert.equal(platformToProto("web"), Platform.UNSPECIFIED);
  assert.equal(platformToProto("windows"), Platform.UNSPECIFIED);
});

test("only Expo's own token shapes are accepted", () => {
  assert.equal(isExpoPushToken("ExponentPushToken[abc123]"), true);
  assert.equal(isExpoPushToken("ExpoPushToken[abc123]"), true);
  assert.equal(isExpoPushToken("abc123"), false);
  assert.equal(isExpoPushToken(""), false);
  assert.equal(isExpoPushToken("ExponentPushToken[]"), false);
});
