import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";

import { decodeAccessClaims, type AccessClaims } from "./claims.ts";
import { SessionManager, type SessionStatus, type TokenStore, type Tokens } from "./session.ts";

export interface AuthContextValue {
  status: SessionStatus;
  /** Resolves a valid access token, refreshing if needed. Null when anonymous. */
  getAccessToken(): Promise<string | null>;
  /** Called after a successful Login/Register RPC. */
  signIn(tokens: Tokens): Promise<void>;
  signOut(): Promise<void>;
  /**
   * Who is signed in, read from the access token's own claims. Null when anonymous.
   *
   * For RENDERING decisions only — it is not verified here (see claims.ts). A screen uses it
   * to decide whether to draw an admin-only control; the server decides whether the resulting
   * request is allowed.
   */
  claims: AccessClaims | null;
  /** Forces a token refresh so claims changed server-side (e.g. new family_id) take effect. */
  refreshNow(): Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export interface AuthProviderProps {
  store: TokenStore;
  /** Calls auth.v1.AuthService/Refresh. Injected so this package never imports the SDK. */
  refresh: (refreshToken: string) => Promise<Tokens>;
  /**
   * Calls auth.v1.AuthService/Logout, which revokes the whole refresh-token chain server-side.
   * Injected for the same reason `refresh` is: this package never imports the SDK.
   *
   * Omitted, sign-out only wipes the tokens from this device and the chain stays valid on the
   * server until it expires on its own — which is what a stolen refresh token needs to keep
   * minting access tokens. Apps should pass it.
   */
  revoke?: (refreshToken: string) => Promise<void>;
  /**
   * Whether a refresh failure means the server rejected the token, as opposed to the request
   * never arriving. Only a rejection ends the session. Injected for the same reason `refresh`
   * is: classifying it needs the RPC library, which this package does not import.
   *
   * Omitted, every failure is treated as a rejection — the behaviour before this existed.
   */
  isRefreshRejection?: (error: unknown) => boolean;
  children: ReactNode;
}

export function AuthProvider({
  store,
  refresh,
  revoke,
  isRefreshRejection,
  children,
}: AuthProviderProps) {
  const manager = useMemo(
    () => new SessionManager(store, refresh, Date.now, isRefreshRejection),
    [store, refresh, isRefreshRejection],
  );
  const [status, setStatus] = useState<SessionStatus>("loading");
  const [claims, setClaims] = useState<AccessClaims | null>(null);

  /**
   * Publishes the manager's state to the tree. Status and claims move together on purpose:
   * they both change on exactly the same events (load, sign-in, refresh, sign-out), and a
   * refresh in particular can hand back a DIFFERENT family_id — that is what `refreshNow`
   * exists for after onboarding. Updating one without the other is how a screen ends up
   * authenticated as one family while rendering another's controls.
   */
  const sync = useCallback(() => {
    setStatus(manager.status());
    setClaims(decodeAccessClaims(manager.current()?.accessToken));
  }, [manager]);

  useEffect(() => {
    let cancelled = false;
    void manager.load().then(() => {
      if (!cancelled) sync();
    });
    return () => {
      cancelled = true;
    };
  }, [manager, sync]);

  const getAccessToken = useCallback(async () => {
    const tokens = await manager.ensureFresh();
    // A refresh failure logs the user out; reflect that in the tree immediately.
    sync();
    return tokens?.accessToken ?? null;
  }, [manager, sync]);

  const signIn = useCallback(
    async (tokens: Tokens) => {
      await manager.set(tokens);
      sync();
    },
    [manager, sync],
  );

  const signOut = useCallback(async () => {
    await manager.end(revoke);
    sync();
  }, [manager, revoke, sync]);

  const refreshNow = useCallback(async () => {
    await manager.forceRefresh();
    sync();
  }, [manager, sync]);

  const value = useMemo<AuthContextValue>(
    () => ({ status, claims, getAccessToken, signIn, signOut, refreshNow }),
    [status, claims, getAccessToken, signIn, signOut, refreshNow],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth: no <AuthProvider> above this component");
  return ctx;
}
