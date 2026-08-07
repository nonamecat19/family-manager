import { ApiProvider } from "@fm/api";
import { AuthProvider, secureTokenStore, tokensFromResponse, useAuth, type Tokens } from "@fm/auth";
import { Loading } from "@fm/ui";
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { AuthService } from "@fm/sdk/auth/v1/auth_pb";
import Constants from "expo-constants";
import { Slot, useRouter, useSegments } from "expo-router";
import { StatusBar } from "expo-status-bar";
import { useCallback, useEffect } from "react";

const API_BASE_URL =
  (Constants.expoConfig?.extra?.apiBaseUrl as string | undefined) ?? "http://localhost:8083";

/**
 * The refresh call is built here with its own transport rather than through @fm/api: the
 * interceptor in @fm/api asks the session for a token, and a refresh that asked the session
 * for a token would recurse.
 */
const refreshClient = createClient(
  AuthService,
  createConnectTransport({ baseUrl: API_BASE_URL, useBinaryFormat: false }),
);

async function refresh(refreshToken: string): Promise<Tokens> {
  const res = await refreshClient.refresh({ refreshToken });
  return tokensFromResponse(res, Date.now());
}

export default function RootLayout() {
  return (
    <AuthProvider store={secureTokenStore} refresh={refresh}>
      <ApiGate />
      <StatusBar style="auto" />
    </AuthProvider>
  );
}

/** Mounts the API provider with a token getter, and routes signed-out users to login. */
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
    <ApiProvider baseUrl={API_BASE_URL} getAccessToken={getToken}>
      <Slot />
    </ApiProvider>
  );
}
