import assert from "node:assert/strict";
import { test } from "node:test";

import { INSTALL_ID_KEY, newInstallId, readOrCreateInstallId, type InstallIdStorage } from "./installId.ts";

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

test("an install id is random hex, different every time it is minted", () => {
  const a = newInstallId();
  const b = newInstallId();
  assert.match(a, /^[0-9a-f]{32}$/);
  assert.notEqual(a, b);
});

test("the install id is minted once and then read back from storage", async () => {
  const storage = memoryStorage();
  const first = await readOrCreateInstallId(storage);
  const second = await readOrCreateInstallId(storage);

  assert.equal(first, second);
  assert.equal(storage.values.get(INSTALL_ID_KEY), first);
});

test("an id already in storage is kept", async () => {
  const storage = memoryStorage();
  storage.values.set(INSTALL_ID_KEY, "kept");
  assert.equal(await readOrCreateInstallId(storage), "kept");
});
