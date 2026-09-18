import "../global.css";

import { ApiProvider, isRefreshRejection, usePushRegistration } from "@fm/api";
import { AuthProvider, secureTokenStore, tokensFromResponse, useAuth, type Tokens } from "@fm/auth";
import { appTheme, BootSplash, ErrorBoundary, ThemeProvider } from "@fm/ui";
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
import { useCallback, useEffect } from "react";

import { bootT } from "../components/i18n/index.tsx";

const extra = Constants.expoConfig?.extra as
  | { apiBaseUrl?: string; serviceUrls?: Record<string, string> }
  | undefined;

const API_BASE_URL = extra?.apiBaseUrl ?? "http://localhost:8088";
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

const theme = { ...appTheme, fieldStyle: "underline" as const };

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
      <ThemeProvider theme={theme}>
        <BootSplash title={bootT("gate.appName")} label={bootT("gate.restoringSession")} />
      </ThemeProvider>
    );
  }

  return (
    <ErrorBoundary
      message={bootT("gate.renderError")}
      onError={(error) => console.error("[tasks] unhandled render error", error)}
    >
      <ThemeProvider theme={theme}>
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

  useEffect(() => {
    if (status === "loading") return;
    const inAuthGroup = segments[0] === "(auth)";

    if (status === "anonymous" && !inAuthGroup) {
      router.replace("/(auth)/login");
    } else if (status === "authenticated" && inAuthGroup) {
      router.replace("/(app)");
    }
  }, [status, segments, router]);

  if (status === "loading") {
    return <BootSplash title={bootT("gate.appName")} label={bootT("gate.restoringSession")} />;
  }

  return (
    <ApiProvider baseUrl={API_BASE_URL} serviceUrls={SERVICE_URLS} getAccessToken={getToken}>
      <PushRegistration />
      <Slot />
    </ApiProvider>
  );
}

function PushRegistration() {
  usePushRegistration({ app: "tasks" });
  return null;
}