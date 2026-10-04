import "../global.css";

import { ApiProvider, createQueryClient, isRefreshRejection, usePushRegistration } from "@fm/api";
import { AuthProvider, secureTokenStore, tokensFromResponse, useAuth, type Tokens } from "@fm/auth";
import { appTheme, ErrorBoundary, Loading, ThemeProvider } from "@fm/ui";
import { Alegreya_800ExtraBold } from "@expo-google-fonts/alegreya/800ExtraBold";
import { NunitoSans_400Regular } from "@expo-google-fonts/nunito-sans/400Regular";
import { NunitoSans_500Medium } from "@expo-google-fonts/nunito-sans/500Medium";
import { NunitoSans_600SemiBold } from "@expo-google-fonts/nunito-sans/600SemiBold";
import { NunitoSans_700Bold } from "@expo-google-fonts/nunito-sans/700Bold";
import { NunitoSans_800ExtraBold } from "@expo-google-fonts/nunito-sans/800ExtraBold";
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { AuthService } from "@fm/sdk/auth/v1/auth_pb";
import Constants from "expo-constants";
import { useFonts } from "expo-font";
import { Slot, useRouter, useSegments } from "expo-router";
import { StatusBar } from "expo-status-bar";
import { useCallback, useEffect, useMemo, useRef } from "react";

import { strings } from "../components/i18n/index.ts";
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

export default function RootLayout() {
  const [fontsLoaded] = useFonts({
    Alegreya_800ExtraBold,
    NunitoSans_400Regular,
    NunitoSans_500Medium,
    NunitoSans_600SemiBold,
    NunitoSans_700Bold,
    NunitoSans_800ExtraBold,
  });

  if (!fontsLoaded) {
    return (
      <ThemeProvider theme={appTheme}>
        <Loading label={strings.app.booting} />
      </ThemeProvider>
    );
  }

  return (
    <ErrorBoundary
      message={strings.app.crashed}
      onError={(error) => console.error("[notes] unhandled render error", error)}
    >
      <ThemeProvider theme={appTheme}>
        <AuthProvider
          store={secureTokenStore}
          refresh={refresh}
          revoke={revoke}
          isRefreshRejection={isRefreshRejection}
        >
          <ApiGate />
          <StatusBar style="dark" />
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

  if (status === "loading") return <Loading label={strings.app.restoring} />;

  return (
    <ApiProvider
      baseUrl={API_BASE_URL}
      serviceUrls={SERVICE_URLS}
      getAccessToken={getToken}
      queryClient={queryClient}
    >
      <PushRegistration />
      <Slot />
    </ApiProvider>
  );
}

function PushRegistration() {
  usePushRegistration({ app: "notes" });
  return null;
}
