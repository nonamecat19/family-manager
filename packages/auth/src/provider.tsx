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
  getAccessToken(): Promise<string | null>;
  signIn(tokens: Tokens): Promise<void>;
  signOut(): Promise<void>;
  claims: AccessClaims | null;
  refreshNow(): Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export interface AuthProviderProps {
  store: TokenStore;
  refresh: (refreshToken: string) => Promise<Tokens>;
  revoke?: (refreshToken: string) => Promise<void>;
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
