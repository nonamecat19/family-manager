/**
 * Session state machine: what tokens we hold, when they expire, and when to refresh.
 *
 * Pure logic on purpose — no React, no SecureStore, no clock of its own. Storage and timers
 * are injected, which is what makes expiry behaviour testable without waiting 15 minutes.
 */

export interface Tokens {
  accessToken: string;
  refreshToken: string;
  /** Absolute expiry of the access token, epoch milliseconds. */
  expiresAt: number;
}

export interface TokenStore {
  read(): Promise<Tokens | null>;
  write(tokens: Tokens): Promise<void>;
  clear(): Promise<void>;
}

export type SessionStatus = "loading" | "authenticated" | "anonymous";

/**
 * Refresh this long before the access token actually expires, so an in-flight request never
 * races the expiry.
 */
export const REFRESH_MARGIN_MS = 60_000;

export function isExpired(tokens: Tokens, now: number): boolean {
  return now >= tokens.expiresAt;
}

/** True when the token is inside the refresh margin (or already past expiry). */
export function needsRefresh(tokens: Tokens, now: number): boolean {
  return now >= tokens.expiresAt - REFRESH_MARGIN_MS;
}

/** Milliseconds until the next refresh should fire; 0 when it is already due. */
export function msUntilRefresh(tokens: Tokens, now: number): number {
  return Math.max(0, tokens.expiresAt - REFRESH_MARGIN_MS - now);
}

/** Builds Tokens from an auth.v1 token response, which carries a relative expires_in. */
export function tokensFromResponse(
  res: { accessToken: string; refreshToken: string; expiresIn: number | bigint },
  now: number,
): Tokens {
  return {
    accessToken: res.accessToken,
    refreshToken: res.refreshToken,
    expiresAt: now + Number(res.expiresIn) * 1000,
  };
}

/**
 * Serialises refreshes: many components can call ensureFresh() at once (a screen mounting
 * fires five queries) and exactly one network refresh happens.
 */
export class SessionManager {
  private tokens: Tokens | null = null;
  private inFlight: Promise<Tokens | null> | null = null;
  private loaded = false;

  private readonly store: TokenStore;
  private readonly refreshFn: (refreshToken: string) => Promise<Tokens>;
  private readonly now: () => number;

  // Written out rather than as parameter properties: Node's type-stripping runtime (used by
  // `pnpm test`) rejects that syntax.
  constructor(
    store: TokenStore,
    refreshFn: (refreshToken: string) => Promise<Tokens>,
    now: () => number = Date.now,
  ) {
    this.store = store;
    this.refreshFn = refreshFn;
    this.now = now;
  }

  /** Loads persisted tokens once. Safe to call repeatedly. */
  async load(): Promise<Tokens | null> {
    if (!this.loaded) {
      this.tokens = await this.store.read();
      this.loaded = true;
    }
    return this.tokens;
  }

  current(): Tokens | null {
    return this.tokens;
  }

  status(): SessionStatus {
    if (!this.loaded) return "loading";
    return this.tokens ? "authenticated" : "anonymous";
  }

  async set(tokens: Tokens): Promise<void> {
    this.tokens = tokens;
    this.loaded = true;
    await this.store.write(tokens);
  }

  async clear(): Promise<void> {
    this.tokens = null;
    this.loaded = true;
    this.inFlight = null;
    await this.store.clear();
  }

  /**
   * Returns a usable access token, refreshing first when it is due. Returns null when there
   * is no session or the refresh failed — the caller then treats the request as anonymous.
   */
  async ensureFresh(): Promise<Tokens | null> {
    await this.load();
    const tokens = this.tokens;
    if (!tokens) return null;
    if (!needsRefresh(tokens, this.now())) return tokens;

    // Collapse concurrent callers onto one refresh.
    this.inFlight ??= this.doRefresh(tokens.refreshToken).finally(() => {
      this.inFlight = null;
    });
    return this.inFlight;
  }

  private async doRefresh(refreshToken: string): Promise<Tokens | null> {
    try {
      const fresh = await this.refreshFn(refreshToken);
      await this.set(fresh);
      return fresh;
    } catch {
      // A failed refresh means the refresh token is spent or revoked: drop the session
      // rather than retrying a token the server has already rejected.
      await this.clear();
      return null;
    }
  }
}
