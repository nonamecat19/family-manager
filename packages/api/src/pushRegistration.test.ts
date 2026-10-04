import assert from "node:assert/strict";
import { test } from "node:test";

import { PushRegistration, type PushRegistrar } from "./pushRegistration.ts";

const TOKEN = "ExponentPushToken[abc]";

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((r) => {
    resolve = r;
  });
  return { promise, resolve };
}

function fakeRegistrar(overrides: Partial<PushRegistrar> = {}) {
  const calls = { registered: [] as string[], unregistered: [] as string[] };
  const registrar: PushRegistrar = {
    resolveToken: async () => TOKEN,
    register: async (token) => {
      calls.registered.push(token);
    },
    unregister: async (token) => {
      calls.unregistered.push(token);
    },
    ...overrides,
  };
  return { registrar, calls };
}

test("signing out unregisters the token this device registered", async () => {
  const { registrar, calls } = fakeRegistrar();
  const reg = new PushRegistration(registrar);
  await reg.sync("u1");
  await reg.signOut();

  assert.deepEqual(calls.registered, [TOKEN]);
  assert.deepEqual(calls.unregistered, [TOKEN]);
});

test("sign-out during an in-flight register waits for it, then unregisters what it registered", async () => {
  const gate = deferred<void>();
  const order: string[] = [];
  const { registrar } = fakeRegistrar({
    register: async () => {
      await gate.promise;
      order.push("register");
    },
    unregister: async () => {
      order.push("unregister");
    },
  });
  const reg = new PushRegistration(registrar);
  void reg.sync("u1");
  await new Promise((r) => setImmediate(r));

  const out = reg.signOut();
  gate.resolve();
  await out;

  assert.deepEqual(order, ["register", "unregister"]);
});

test("sign-out while the push token is still being resolved registers nothing", async () => {
  const token = deferred<string | null>();
  const { registrar, calls } = fakeRegistrar({ resolveToken: () => token.promise });
  const reg = new PushRegistration(registrar);
  void reg.sync("u1");

  const out = reg.signOut();
  token.resolve(TOKEN);
  await out;

  assert.deepEqual(calls.registered, [], "nothing may be registered once sign-out began");
  assert.deepEqual(calls.unregistered, []);
});

test("the same user and token are registered once", async () => {
  const { registrar, calls } = fakeRegistrar();
  const reg = new PushRegistration(registrar);
  await reg.sync("u1");
  await reg.sync("u1");
  assert.deepEqual(calls.registered, [TOKEN]);

  await reg.sync("u2");
  assert.deepEqual(calls.registered, [TOKEN, TOKEN], "a different account registers again");
});

test("a failed unregister surfaces to the auth layer but the token is forgotten", async () => {
  const { registrar, calls } = fakeRegistrar({
    unregister: async () => {
      throw new Error("offline");
    },
  });
  const reg = new PushRegistration(registrar);
  await reg.sync("u1");
  await assert.rejects(reg.signOut(), /offline/);

  await reg.sync("u1");
  assert.deepEqual(calls.registered, [TOKEN, TOKEN], "the next sign-in registers afresh");
});

test("a failed register is retried on the next sync and still unregistered on sign-out", async () => {
  let attempts = 0;
  const { registrar, calls } = fakeRegistrar({
    register: async () => {
      attempts++;
      if (attempts === 1) throw new Error("503");
    },
  });
  const reg = new PushRegistration(registrar);
  await reg.sync("u1");
  await reg.sync("u1");
  assert.equal(attempts, 2);

  await reg.signOut();
  assert.deepEqual(calls.unregistered, [TOKEN]);
});
