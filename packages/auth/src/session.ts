export interface Tokens {
  accessToken: string;
  refreshToken: string;
  expiresAt: number;
}

export interface TokenStore {
  read(): Promise<Tokens | null>;
  write(tokens: Tokens): Promise<void>;
  clear(): Promise<void>;
}

export type SessionStatus = "loading" | "authenticated" | "anonymous";

export const REFRESH_MARGIN_MS = 60_000;

export function isExpired(tokens: Tokens, now: number): boolean {
  return now >= tokens.expiresAt;
}

export function needsRefresh(tokens: Tokens, now: number): boolean {
  return now >= tokens.expiresAt - REFRESH_MARGIN_MS;
}

export function msUntilRefresh(tokens: Tokens, now: number): number {
  return Math.max(0, tokens.expiresAt - REFRESH_MARGIN_MS - now);
}

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

export class SessionManager {
  private tokens: Tokens | null = null;
  private inFlight: Promise<Tokens | null> | null = null;
  private loaded = false;

  private readonly store: TokenStore;
  private readonly refreshFn: (refreshToken: string) => Promise<Tokens>;
  private readonly now: () => number;
  private readonly isRejection: (error: unknown) => boolean;

  constructor(
    store: TokenStore,
    refreshFn: (refreshToken: string) => Promise<Tokens>,
    now: () => number = Date.now,
    isRejection: (error: unknown) => boolean = () => true,
  ) {
    this.store = store;
    this.refreshFn = refreshFn;
    this.now = now;
    this.isRejection = isRejection;
  }

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

  async end(revoke?: (refreshToken: string) => Promise<void>): Promise<void> {
    const tokens = this.tokens;
    if (revoke && tokens) {
      try {
        await revoke(tokens.refreshToken);
      } catch (error) {
        void error;
      }
    }
    await this.clear();
  }

  async ensureFresh(): Promise<Tokens | null> {
    await this.load();
    const tokens = this.tokens;
    if (!tokens) return null;
    if (!needsRefresh(tokens, this.now())) return tokens;

    this.inFlight ??= this.doRefresh(tokens.refreshToken).finally(() => {
      this.inFlight = null;
    });
    return this.inFlight;
  }

  async forceRefresh(): Promise<Tokens | null> {
    await this.load();
    const tokens = this.tokens;
    if (!tokens) return null;

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
    } catch (error) {
      if (this.isRejection(error)) {
        await this.clear();
      }
      return null;
    }
  }
}
