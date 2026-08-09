import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";

import { SessionManager, type SessionStatus, type TokenStore, type Tokens } from "./session.ts";

export interface AuthContextValue {
  status: SessionStatus;
  /** Resolves a valid access token, refreshing if needed. Null when anonymous. */
  getAccessToken(): Promise<string | null>;
  /** Called after a successful Login/Register RPC. */
  signIn(tokens: Tokens): Promise<void>;
  signOut(): Promise<void>;
  /** Forces a token refresh so claims changed server-side (e.g. new family_id) take effect. */
  refreshNow(): Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export interface AuthProviderProps {
  store: TokenStore;
  /** Calls auth.v1.AuthService/Refresh. Injected so this package never imports the SDK. */
  refresh: (refreshToken: string) => Promise<Tokens>;
  children: ReactNode;
}

export function AuthProvider({ store, refresh, children }: AuthProviderProps) {
  const manager = useMemo(() => new SessionManager(store, refresh), [store, refresh]);
  const [status, setStatus] = useState<SessionStatus>("loading");

  useEffect(() => {
    let cancelled = false;
    void manager.load().then(() => {
      if (!cancelled) setStatus(manager.status());
    });
    return () => {
      cancelled = true;
    };
  }, [manager]);

  const getAccessToken = useCallback(async () => {
    const tokens = await manager.ensureFresh();
    // A refresh failure logs the user out; reflect that in the tree immediately.
    setStatus(manager.status());
    return tokens?.accessToken ?? null;
  }, [manager]);

  const signIn = useCallback(
    async (tokens: Tokens) => {
      await manager.set(tokens);
      setStatus(manager.status());
    },
    [manager],
  );

  const signOut = useCallback(async () => {
    await manager.clear();
    setStatus(manager.status());
  }, [manager]);

  const refreshNow = useCallback(async () => {
    await manager.forceRefresh();
    setStatus(manager.status());
  }, [manager]);

  const value = useMemo<AuthContextValue>(
    () => ({ status, getAccessToken, signIn, signOut, refreshNow }),
    [status, getAccessToken, signIn, signOut, refreshNow],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth: no <AuthProvider> above this component");
  return ctx;
}
