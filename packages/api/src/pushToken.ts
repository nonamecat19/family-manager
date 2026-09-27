import type { App, Platform } from "@fm/sdk/notifications/v1/notifications_pb";

export type PushApp = "notes" | "finance" | "recipes";

const APP_BY_NAME: Record<PushApp, App> = {
  notes: 1 as App,
  finance: 2 as App,
  recipes: 3 as App,
};

const PLATFORM_UNSPECIFIED = 0 as Platform;
const PLATFORM_IOS = 1 as Platform;
const PLATFORM_ANDROID = 2 as Platform;

export function appToProto(app: PushApp): App {
  return APP_BY_NAME[app];
}

export function platformToProto(os: string): Platform {
  if (os === "ios") return PLATFORM_IOS;
  if (os === "android") return PLATFORM_ANDROID;
  return PLATFORM_UNSPECIFIED;
}

export function isExpoPushToken(token: string): boolean {
  return /^Expo(nent)?PushToken\[.+\]$/.test(token);
}
