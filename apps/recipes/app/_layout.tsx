import "../global.css";

import { ApiProvider, isRefreshRejection } from "@fm/api";
import { AuthProvider, secureTokenStore, tokensFromResponse, useAuth, type Tokens } from "@fm/auth";
import { ErrorBoundary, Loading, ThemeProvider } from "@fm/ui";
import { organicTheme, type Theme } from "@fm/theme";
import { Alegreya_800ExtraBold } from "@expo-google-fonts/alegreya/800ExtraBold";
import { NunitoSans_400Regular } from "@expo-google-fonts/nunito-sans/400Regular";
import { NunitoSans_500Medium } from "@expo-google-fonts/nunito-sans/500Medium";
import { NunitoSans_600SemiBold } from "@expo-google-fonts/nunito-sans/600SemiBold";
import { NunitoSans_700Bold } from "@expo-google-fonts/nunito-sans/700Bold";
import { NunitoSans_800ExtraBold } from "@expo-google-fonts/nunito-sans/800ExtraBold";
import { useFonts } from "expo-font";

import { bootT } from "../components/i18n/index.tsx";
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { AuthService } from "@fm/sdk/auth/v1/auth_pb";
import Constants from "expo-constants";
import { Slot, useRouter, useSegments } from "expo-router";
import { StatusBar } from "expo-status-bar";
import { useCallback, useEffect } from "react";

const extra = Constants.expoConfig?.extra as
  | { apiBaseUrl?: string; serviceUrls?: Record<string, string> }
  | undefined;

const API_BASE_URL = extra?.apiBaseUrl ?? "http://localhost:8084";
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
  ...organicTheme,
  fonts: {
    body: "NunitoSans_400Regular",
    medium: "NunitoSans_500Medium",
    semibold: "NunitoSans_600SemiBold",
    display: "Alegreya_800ExtraBold",
  },
};

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
        <Loading label={bootT("kitchen.warmingOven")} />
      </ThemeProvider>
    );
  }

  return (
    <ErrorBoundary
      message={bootT("kitchen.somethingBurned")}
      onError={(error) => console.error("[recipes] unhandled render error", error)}
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

  if (status === "loading") return <Loading label={bootT("kitchen.restoringSession")} />;

  return (
    <ApiProvider baseUrl={API_BASE_URL} serviceUrls={SERVICE_URLS} getAccessToken={getToken}>
      <Slot />
    </ApiProvider>
  );
}
