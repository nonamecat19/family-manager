import * as SecureStore from "expo-secure-store";

import type { TokenStore, Tokens } from "./session.ts";

const KEY = "fm.session";

export const secureTokenStore: TokenStore = {
  async read(): Promise<Tokens | null> {
    const raw = await SecureStore.getItemAsync(KEY);
    if (!raw) return null;
    try {
      const parsed = JSON.parse(raw) as Partial<Tokens>;
      if (
        typeof parsed.accessToken !== "string" ||
        typeof parsed.refreshToken !== "string" ||
        typeof parsed.expiresAt !== "number"
      ) {
        await SecureStore.deleteItemAsync(KEY);
        return null;
      }
      return parsed as Tokens;
    } catch {
      await SecureStore.deleteItemAsync(KEY);
      return null;
    }
  },

  async write(tokens: Tokens): Promise<void> {
    await SecureStore.setItemAsync(KEY, JSON.stringify(tokens));
  },

  async clear(): Promise<void> {
    await SecureStore.deleteItemAsync(KEY);
  },
};
