import "../global.css";

import { ApiProvider, isRefreshRejection } from "@fm/api";
import { AuthProvider, secureTokenStore, tokensFromResponse, useAuth, type Tokens } from "@fm/auth";
import { ErrorBoundary, ThemeProvider } from "@fm/ui";
import { nocturneTheme, type Theme } from "@fm/theme";
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

async function revoke(refreshToken: string): Promise<void> {
  await refreshClient.logout({ refreshToken });
}

const theme: Theme = { ...nocturneTheme, fieldStyle: "underline" };

export default function RootLayout() {
  return (
    <ErrorBoundary
      message={bootT("gate.renderError")}
      onError={(error) => console.error("[finance] unhandled render error", error)}
    >
      {
}
      <ThemeProvider theme={theme}>
        <AuthProvider store={secureTokenStore} refresh={refresh}
          revoke={revoke} isRefreshRejection={isRefreshRejection}>
        <ApiGate />
        {}
        <StatusBar style="light" />
        </AuthProvider>
      </ThemeProvider>
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
