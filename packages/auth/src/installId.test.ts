import assert from "node:assert/strict";
import { randomBytes as nodeRandomBytes } from "node:crypto";
import { test } from "node:test";

import {
  INSTALL_ID_KEY,
  newInstallId,
  readOrCreateInstallId,
  type InstallIdStorage,
  type RandomBytes,
} from "./installId.ts";

const secure: RandomBytes = (count) => new Uint8Array(nodeRandomBytes(count));

function memoryStorage(): InstallIdStorage & { values: Map<string, string> } {
  const values = new Map<string, string>();
  return {
    values,
    async getItemAsync(key) {
      return values.get(key) ?? null;
    },
    async setItemAsync(key, value) {
      values.set(key, value);
    },
  };
}

test("an install id is 16 random bytes in hex, different every time it is minted", () => {
  const a = newInstallId(secure);
  const b = newInstallId(secure);
  assert.match(a, /^[0-9a-f]{32}$/);
  assert.notEqual(a, b);
});

test("without a secure random source there is no install id, never a weak one", () => {
  assert.equal(
    newInstallId(() => {
      throw new Error("native module missing");
    }),
    "",
  );
  assert.equal(newInstallId(() => new Uint8Array(4)), "");
});

test("the install id is minted once and then read back from storage", async () => {
  const storage = memoryStorage();
  const first = await readOrCreateInstallId(storage, secure);
  const second = await readOrCreateInstallId(storage, secure);

  assert.equal(first, second);
  assert.equal(storage.values.get(INSTALL_ID_KEY), first);
});

test("an id already in storage is kept", async () => {
  const storage = memoryStorage();
  storage.values.set(INSTALL_ID_KEY, "kept");
  assert.equal(await readOrCreateInstallId(storage, secure), "kept");
});

test("a failed mint persists nothing, so a later launch can still mint one", async () => {
  const storage = memoryStorage();
  const id = await readOrCreateInstallId(storage, () => {
    throw new Error("native module missing");
  });
  assert.equal(id, "");
  assert.equal(storage.values.has(INSTALL_ID_KEY), false);
});
