import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { getInstallId, useAuth } from "@fm/auth";
import { useEffect, useRef } from "react";
import { Platform as RNPlatform } from "react-native";

import type { App, Platform } from "@fm/sdk/notifications/v1/notifications_pb";

import { useClients } from "./provider.tsx";
import { queryKeys } from "./queryKeys.ts";
import { unregisterOnSignOut } from "./pushSignOut.ts";
import { appToProto, isExpoPushToken, platformToProto, type PushApp } from "./pushToken.ts";

export interface RegisterPushTokenInput {
  token: string;
  platform: Platform;
  app: App;
  deviceId?: string;
}

export function useRegisterPushToken() {
  const { notifications } = useClients();
  return useMutation({
    mutationFn: (input: RegisterPushTokenInput) =>
      notifications.registerPushToken({
        token: input.token,
        platform: input.platform,
        app: input.app,
        deviceId: input.deviceId ?? "",
      }),
  });
}

export function useUnregisterPushToken() {
  const { notifications } = useClients();
  return useMutation({
    mutationFn: (token: string) => notifications.unregisterPushToken({ token }),
  });
}

export function useNotificationPreferences(opts: { enabled?: boolean } = {}) {
  const { notifications } = useClients();
  return useQuery({
    queryKey: queryKeys.notificationPreferences(),
    queryFn: () => notifications.getPreferences({}),
    enabled: opts.enabled ?? true,
  });
}

export function useSetNotificationPreferences() {
  const { notifications } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (muted: string[]) => (await notifications.setPreferences({ muted })).muted,
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.notificationPreferences() }),
  });
}

async function resolveExpoPushToken(): Promise<string | null> {
  if (RNPlatform.OS === "web") return null;

  const [Notifications, { default: Constants }] = await Promise.all([
    import("expo-notifications"),
    import("expo-constants"),
  ]);

  const projectId =
    (Constants.expoConfig?.extra as { eas?: { projectId?: string } } | undefined)?.eas?.projectId ??
    Constants.easConfig?.projectId;
  if (!projectId) return null;

  const existing = await Notifications.getPermissionsAsync();
  let finalStatus = existing.status;
  if (finalStatus !== "granted") {
    const requested = await Notifications.requestPermissionsAsync();
    finalStatus = requested.status;
  }
  if (finalStatus !== "granted") return null;

  const response = await Notifications.getExpoPushTokenAsync({ projectId });
  return isExpoPushToken(response.data) ? response.data : null;
}

export function usePushRegistration({ app }: { app: PushApp }): void {
  const { status, claims, registerBeforeSignOut } = useAuth();
  const { notifications } = useClients();
  const register = useRegisterPushToken();
  const registeredKey = useRef<string | null>(null);
  const registeredToken = useRef<string | null>(null);

  useEffect(() => {
    if (RNPlatform.OS === "web") return;
    return unregisterOnSignOut(
      registerBeforeSignOut,
      () => registeredToken.current,
      (token) => notifications.unregisterPushToken({ token }),
      () => {
        registeredToken.current = null;
        registeredKey.current = null;
      },
    );
  }, [registerBeforeSignOut, notifications]);

  useEffect(() => {
    if (RNPlatform.OS === "web") return;

    if (status !== "authenticated") {
      registeredToken.current = null;
      registeredKey.current = null;
      return;
    }

    const userId = claims?.userId ?? "";
    let cancelled = false;

    void (async () => {
      const token = await resolveExpoPushToken();
      if (!token || cancelled) return;

      const key = `${userId}:${token}`;
      if (registeredKey.current === key) return;

      const deviceId = await getInstallId().catch(() => "");
      if (cancelled) return;

      registeredKey.current = key;
      registeredToken.current = token;
      register.mutate({
        token,
        platform: platformToProto(RNPlatform.OS),
        app: appToProto(app),
        deviceId,
      });
    })();

    return () => {
      cancelled = true;
    };
  }, [status, claims?.userId, app]);
}

export type { Topic, GetPreferencesResponse } from "@fm/sdk/notifications/v1/notifications_pb";
