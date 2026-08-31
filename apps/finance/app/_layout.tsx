import "../global.css";

import { ApiProvider, isRefreshRejection } from "@fm/api";
import { AuthProvider, secureTokenStore, tokensFromResponse, useAuth, type Tokens } from "@fm/auth";
import { ErrorBoundary } from "@fm/ui";
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { AuthService } from "@fm/sdk/auth/v1/auth_pb";
import Constants from "expo-constants";
import { Slot, useRouter, useSegments } from "expo-router";
import { StatusBar } from "expo-status-bar";
import { useCallback, useEffect } from "react";
import { Text, View } from "react-native";

import { bootT } from "../components/i18n/index.tsx";
import { nocturne } from "../components/nocturne/index.ts";

const extra = Constants.expoConfig?.extra as
  | { apiBaseUrl?: string; serviceUrls?: Record<string, string> }
  | undefined;

const API_BASE_URL = extra?.apiBaseUrl ?? "http://localhost:8083";
const SERVICE_URLS = extra?.serviceUrls;

// The only place in the app that touches the generated SDK directly: refreshing a session
// happens outside the ApiProvider it would otherwise authenticate.
const refreshClient = createClient(
  AuthService,
  createConnectTransport({
    baseUrl: SERVICE_URLS?.auth ?? API_BASE_URL,
    useBinaryFormat: false,
  }),
);

async function refresh(refreshToken: string): Promise<Tokens> {
  const res = await refreshClient.refresh({ refreshToken });
  return tokensFromResponse(res, Date.now());
}

export default function RootLayout() {
  return (
    // Outside AuthProvider, so a crash while restoring the session is caught too — which is
    // the one place a user cannot navigate away from.
    <ErrorBoundary
      message={bootT("gate.renderError")}
      onError={(error) => console.error("[finance] unhandled render error", error)}
    >
      <AuthProvider store={secureTokenStore} refresh={refresh} isRefreshRejection={isRefreshRejection}>
        <ApiGate />
        {/* Nocturne is dark-only, so the status bar is light on every screen, always. */}
        <StatusBar style="light" />
      </AuthProvider>
    </ErrorBoundary>
  );
}

function ApiGate() {
  const { status, getAccessToken } = useAuth();
  const segments = useSegments();
  const router = useRouter();

  const getToken = useCallback(() => getAccessToken(), [getAccessToken]);

  useEffect(() => {
    if (status === "loading") return;
    const inAuthGroup = segments[0] === "(auth)";

    if (status === "anonymous" && !inAuthGroup) {
      router.replace("/(auth)/login");
    } else if (status === "authenticated" && inAuthGroup) {
      router.replace("/(app)");
    }
  }, [status, segments, router]);

  if (status === "loading") return <BootScreen label={bootT("gate.restoringSession")} />;

  return (
    <ApiProvider baseUrl={API_BASE_URL} serviceUrls={SERVICE_URLS} getAccessToken={getToken}>
      <Slot />
    </ApiProvider>
  );
}

/**
 * The pre-provider loading screen. Painted from the raw tokens rather than `@fm/ui`'s
 * `Loading`, whose light surfaces would flash white before the first Nocturne screen mounts.
 * Its label comes from `bootT`, the device-locale lookup — `I18nProvider` cannot exist this
 * early.
 */
function BootScreen({ label }: { label: string }) {
  return (
    <View className="flex-1 items-center justify-center gap-n3" style={{ backgroundColor: nocturne.bg }}>
      <Text className="text-[20px] font-medium" style={{ color: nocturne.text }}>
        {bootT("gate.appName")}
      </Text>
      <Text className="text-[13px]" style={{ color: nocturne.neutral[500] }}>
        {label}
      </Text>
    </View>
  );
}
