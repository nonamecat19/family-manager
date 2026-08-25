import "../global.css";

import { ApiProvider } from "@fm/api";
import { AuthProvider, secureTokenStore, tokensFromResponse, useAuth, type Tokens } from "@fm/auth";
import { Loading } from "@fm/ui";
// Imported per weight rather than from the package root: the root index requires every
// weight and italic, and Metro bundles what it sees — ~500 kB of TTFs the app never renders.
import { Caprasimo_400Regular } from "@expo-google-fonts/caprasimo/400Regular";
import { Figtree_400Regular } from "@expo-google-fonts/figtree/400Regular";
import { Figtree_500Medium } from "@expo-google-fonts/figtree/500Medium";
import { Figtree_600SemiBold } from "@expo-google-fonts/figtree/600SemiBold";
import { Figtree_700Bold } from "@expo-google-fonts/figtree/700Bold";
import { Figtree_800ExtraBold } from "@expo-google-fonts/figtree/800ExtraBold";
import { useFonts } from "expo-font";
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

export default function RootLayout() {
  // Organic is a two-face system — Caprasimo for display, Figtree for everything else — and
  // every weight is a separate file, so the whole set is loaded up front rather than letting
  // screens render in the platform font and reflow a frame later.
  const [fontsLoaded] = useFonts({
    Caprasimo_400Regular,
    Figtree_400Regular,
    Figtree_500Medium,
    Figtree_600SemiBold,
    Figtree_700Bold,
    Figtree_800ExtraBold,
  });

  if (!fontsLoaded) return <Loading label="Warming the oven…" />;

  return (
    <AuthProvider store={secureTokenStore} refresh={refresh}>
      <ApiGate />
      <StatusBar style="dark" />
    </AuthProvider>
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

  if (status === "loading") return <Loading label="Restoring your session…" />;

  return (
    <ApiProvider baseUrl={API_BASE_URL} serviceUrls={SERVICE_URLS} getAccessToken={getToken}>
      <Slot />
    </ApiProvider>
  );
}