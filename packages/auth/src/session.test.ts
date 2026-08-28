import assert from "node:assert/strict";
import { test } from "node:test";

import {
  REFRESH_MARGIN_MS,
  SessionManager,
  isExpired,
  msUntilRefresh,
  needsRefresh,
  tokensFromResponse,
  type TokenStore,
  type Tokens,
} from "./session.ts";

function memoryStore(initial: Tokens | null = null): TokenStore & { value: Tokens | null } {
  return {
    value: initial,
    async read() {
      return this.value;
    },
    async write(t: Tokens) {
      this.value = t;
    },
    async clear() {
      this.value = null;
    },
  };
}

const NOW = 1_800_000_000_000;
const fresh: Tokens = { accessToken: "a", refreshToken: "r", expiresAt: NOW + 15 * 60_000 };

test("expiry predicates", () => {
  assert.equal(isExpired(fresh, NOW), false);
  assert.equal(isExpired(fresh, fresh.expiresAt), true);
  assert.equal(needsRefresh(fresh, NOW), false);
  assert.equal(needsRefresh(fresh, fresh.expiresAt - REFRESH_MARGIN_MS), true);
  assert.equal(needsRefresh(fresh, fresh.expiresAt + 1), true);
});

test("msUntilRefresh never goes negative", () => {
  assert.equal(msUntilRefresh(fresh, NOW), 15 * 60_000 - REFRESH_MARGIN_MS);
  assert.equal(msUntilRefresh(fresh, fresh.expiresAt), 0);
});

test("tokensFromResponse turns expires_in into an absolute expiry", () => {
  const t = tokensFromResponse({ accessToken: "a", refreshToken: "r", expiresIn: 900 }, NOW);
  assert.equal(t.expiresAt, NOW + 900_000);
  // The Go side sends int64, which protobuf-es surfaces as bigint.
  const big = tokensFromResponse({ accessToken: "a", refreshToken: "r", expiresIn: 900n }, NOW);
  assert.equal(big.expiresAt, NOW + 900_000);
});

test("status reflects load and clear", async () => {
  const store = memoryStore(fresh);
  const mgr = new SessionManager(store, async () => fresh, () => NOW);

  assert.equal(mgr.status(), "loading");
  await mgr.load();
  assert.equal(mgr.status(), "authenticated");

  await mgr.clear();
  assert.equal(mgr.status(), "anonymous");
  assert.equal(store.value, null);
});

test("ensureFresh returns the token untouched when it is not due", async () => {
  let refreshes = 0;
  const mgr = new SessionManager(
    memoryStore(fresh),
    async () => {
      refreshes++;
      return fresh;
    },
    () => NOW,
  );

  assert.deepEqual(await mgr.ensureFresh(), fresh);
  assert.equal(refreshes, 0);
});

test("ensureFresh refreshes inside the margin and persists the result", async () => {
  const store = memoryStore({ ...fresh, expiresAt: NOW + 30_000 });
  const rotated: Tokens = { accessToken: "a2", refreshToken: "r2", expiresAt: NOW + 900_000 };

  const seen: string[] = [];
  const mgr = new SessionManager(
    store,
    async (rt) => {
      seen.push(rt);
      return rotated;
    },
    () => NOW,
  );

  assert.deepEqual(await mgr.ensureFresh(), rotated);
  assert.deepEqual(seen, ["r"]);
  assert.deepEqual(store.value, rotated);
});

test("concurrent ensureFresh calls collapse onto one refresh", async () => {
  const store = memoryStore({ ...fresh, expiresAt: NOW + 30_000 });
  const rotated: Tokens = { accessToken: "a2", refreshToken: "r2", expiresAt: NOW + 900_000 };

  let refreshes = 0;
  const mgr = new SessionManager(
    store,
    async () => {
      refreshes++;
      await new Promise((r) => setTimeout(r, 10));
      return rotated;
    },
    () => NOW,
  );

  const results = await Promise.all([mgr.ensureFresh(), mgr.ensureFresh(), mgr.ensureFresh()]);
  assert.equal(refreshes, 1);
  for (const r of results) assert.deepEqual(r, rotated);
});

test("a failed refresh drops the session instead of retrying a dead token", async () => {
  const store = memoryStore({ ...fresh, expiresAt: NOW + 30_000 });
  const mgr = new SessionManager(
    store,
    async () => {
      throw new Error("401 revoked");
    },
    () => NOW,
  );

  assert.equal(await mgr.ensureFresh(), null);
  assert.equal(store.value, null);
  assert.equal(mgr.status(), "anonymous");
});

test("ensureFresh on an empty store is anonymous, not an error", async () => {
  const mgr = new SessionManager(memoryStore(null), async () => fresh, () => NOW);
  assert.equal(await mgr.ensureFresh(), null);
});

/** Tokens that are inside the refresh margin, so any call triggers a refresh. */
function dueTokens(): Tokens {
  return { accessToken: "a", refreshToken: "r", expiresAt: NOW + REFRESH_MARGIN_MS / 2 };
}

// The failure this distinction exists for: no signal on a train, a captive portal, the VPS
// restarting. None of them says anything about the refresh token, and signing the user out for
// one makes them type their password to get back into an app they never left.
test("a refresh that never reached the server keeps the session", async () => {
  const store = memoryStore(dueTokens());
  const manager = new SessionManager(
    store,
    () => Promise.reject(new Error("network request failed")),
    () => NOW,
    () => false,
  );

  assert.equal(await manager.ensureFresh(), null);
  assert.notEqual(store.value, null, "tokens were discarded for a transport failure");
  assert.equal(manager.status(), "authenticated");
});

test("a rejected refresh token ends the session", async () => {
  const store = memoryStore(dueTokens());
  const manager = new SessionManager(
    store,
    () => Promise.reject(new Error("invalid refresh token")),
    () => NOW,
    () => true,
  );

  assert.equal(await manager.ensureFresh(), null);
  assert.equal(store.value, null, "a rejected token was kept");
  assert.equal(manager.status(), "anonymous");
});

// The default is the behaviour that existed before the classifier did, so a caller that does
// not pass one is not silently changed.
test("without a classifier every failure ends the session", async () => {
  const store = memoryStore(dueTokens());
  const manager = new SessionManager(store, () => Promise.reject(new Error("boom")), () => NOW);

  assert.equal(await manager.ensureFresh(), null);
  assert.equal(store.value, null);
});

// Keeping the tokens is only useful if the next attempt actually retries.
test("a kept session refreshes again on the next call", async () => {
  const store = memoryStore(dueTokens());
  let attempts = 0;
  const manager = new SessionManager(
    store,
    () => {
      attempts++;
      if (attempts === 1) return Promise.reject(new Error("network request failed"));
      return Promise.resolve({ accessToken: "a2", refreshToken: "r2", expiresAt: NOW + 900_000 });
    },
    () => NOW,
    () => false,
  );

  assert.equal(await manager.ensureFresh(), null);
  const fresh = await manager.ensureFresh();
  assert.equal(fresh?.accessToken, "a2");
  assert.equal(attempts, 2);
});
