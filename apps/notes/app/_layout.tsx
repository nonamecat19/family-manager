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

/** Ends the session server-side: Logout revokes the whole refresh-token chain, so the token
 * this device is about to forget cannot go on minting access tokens. */
async function revoke(refreshToken: string): Promise<void> {
  await refreshClient.logout({ refreshToken });
}

/**
 * Nocturne, in Inter. The palette is shared with apps/finance; the faces are not — finance
 * ships no font file and draws in the platform sans. That is exactly why fonts live on the
 * theme rather than on the design system.
 */
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
  // Nocturne is one face in four weights, each its own TTF. Loaded up front rather than per
  // screen: a weight that arrives late reflows the note the user is already reading.
  const [fontsLoaded] = useFonts({
    Inter_400Regular,
    Inter_500Medium,
    Inter_600SemiBold,
    Inter_700Bold,
  });

  if (!fontsLoaded) return <Booting label={strings.app.booting} />;

  return (
    // Outside AuthProvider, so a crash while restoring the session is caught too — the one
    // place the user cannot navigate away from.
    <ErrorBoundary
      message={strings.app.crashed}
      onError={(error) => console.error("[notes] unhandled render error", error)}
    >
      {/* Nocturne is this app's theme; the same shared components render in Organic over in
          apps/recipes without either screen knowing which. */}
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

  // The app owns the QueryClient rather than letting ApiProvider mint one, for a single
  // reason: something has to be able to EMPTY it. This gate keeps ApiProvider mounted when the
  // session ends — the tree below it swaps to the login screen, the provider does not unmount —
  // so without this the cache outlived the session, and with `gcTime` at 24h and note query
  // keys carrying no user segment, the next person to sign in on the device would be served the
  // previous person's notes until the first refetch landed. On a family's shared tablet those
  // are somebody else's private notes on screen. (@fm/api's provider is shared with
  // app:finance and app:recipes, so the fix belongs here, not there.)
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
      // Cancel first: a request that was already in flight for the old session would otherwise
      // land after clear() and repopulate the cache it was just emptied from.
      try {
        await queryClient.cancelQueries();
      } catch (error) {
        console.warn("[notes] could not cancel in-flight queries on sign-out", error);
      }
      queryClient.clear();
      // The offline capture queue is per-user and lives on disk; this forgets it in memory.
      // The signed-out user's own file stays where it is, so their unsent writing comes back
      // when they sign in again — see components/offline/queue.ts.
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

/**
 * The pre-font loading screen. It cannot use @fm/ui's Loading, because that renders text in a
 * font family this app has not loaded yet — the one screen where the platform face is right.
 */
function Booting({ label }: { label: string }) {
  return (
    <View className="flex-1 items-center justify-center gap-[10px] bg-bg">
      <ActivityIndicator color={nocturne.accent.DEFAULT} />
      <Text style={{ color: nocturne.neutral[500], fontSize: 13 }}>{label}</Text>
    </View>
  );
}
