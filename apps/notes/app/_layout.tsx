import "../global.css";

import { Inter_400Regular } from "@expo-google-fonts/inter/400Regular";
import { Inter_500Medium } from "@expo-google-fonts/inter/500Medium";
import { Inter_600SemiBold } from "@expo-google-fonts/inter/600SemiBold";
import { Inter_700Bold } from "@expo-google-fonts/inter/700Bold";
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { ApiProvider, createQueryClient, isRefreshRejection } from "@fm/api";
import { AuthProvider, secureTokenStore, tokensFromResponse, useAuth, type Tokens } from "@fm/auth";
import { AuthService } from "@fm/sdk/auth/v1/auth_pb";
import { ErrorBoundary, ThemeProvider } from "@fm/ui";
import { nocturneTheme, type Theme } from "@fm/theme";
import Constants from "expo-constants";
import { useFonts } from "expo-font";
import { Slot, useRouter, useSegments } from "expo-router";
import { StatusBar } from "expo-status-bar";
import { useCallback, useEffect, useMemo, useRef } from "react";
import { ActivityIndicator, Text, View } from "react-native";

import { strings } from "../components/i18n/index.ts";
import { nocturne } from "../components/nocturne/index.ts";
import { resetCaptureQueue } from "../components/offline/index.ts";

const extra = Constants.expoConfig?.extra as
  | { apiBaseUrl?: string; serviceUrls?: Record<string, string> }
  | undefined;

const API_BASE_URL = extra?.apiBaseUrl ?? "http://localhost:8085";
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

const theme: Theme = {
  ...nocturneTheme,
  fonts: {
    body: "Inter_400Regular",
    medium: "Inter_500Medium",
    semibold: "Inter_600SemiBold",
    display: "Inter_600SemiBold",
  },
};

export default function RootLayout() {
  const [fontsLoaded] = useFonts({
    Inter_400Regular,
    Inter_500Medium,
    Inter_600SemiBold,
    Inter_700Bold,
  });

  if (!fontsLoaded) return <Booting label={strings.app.booting} />;

  return (
    <ErrorBoundary
      message={strings.app.crashed}
      onError={(error) => console.error("[notes] unhandled render error", error)}
    >
      {
}
      <ThemeProvider theme={theme}>
        <AuthProvider
          store={secureTokenStore}
          refresh={refresh}
          revoke={revoke}
          isRefreshRejection={isRefreshRejection}
        >
          <ApiGate />
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

  const queryClient = useMemo(() => createQueryClient(), []);
  const wasAuthenticated = useRef(false);

  useEffect(() => {
    if (status === "authenticated") {
      wasAuthenticated.current = true;
      return;
    }
    if (status !== "anonymous" || !wasAuthenticated.current) return;
    wasAuthenticated.current = false;

    void (async () => {
      try {
        await queryClient.cancelQueries();
      } catch (error) {
        console.warn("[notes] could not cancel in-flight queries on sign-out", error);
      }
      queryClient.clear();
      await resetCaptureQueue();
    })();
  }, [status, queryClient]);

  useEffect(() => {
    if (status === "loading") return;
    const inAuthGroup = segments[0] === "(auth)";

    if (status === "anonymous" && !inAuthGroup) {
      router.replace("/(auth)/login");
    } else if (status === "authenticated" && inAuthGroup) {
      router.replace("/(app)");
    }
  }, [status, segments, router]);

  if (status === "loading") return <Booting label={strings.app.restoring} />;

  return (
    <ApiProvider
      baseUrl={API_BASE_URL}
      serviceUrls={SERVICE_URLS}
      getAccessToken={getToken}
      queryClient={queryClient}
    >
      <Slot />
    </ApiProvider>
  );
}

function Booting({ label }: { label: string }) {
  return (
    <View className="flex-1 items-center justify-center gap-[10px] bg-bg">
      <ActivityIndicator color={nocturne.accent.DEFAULT} />
      <Text style={{ color: nocturne.neutral[500], fontSize: 13 }}>{label}</Text>
    </View>
  );
}
