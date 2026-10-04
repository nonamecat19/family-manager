import assert from "node:assert/strict";
import { test } from "node:test";

import { unregisterOnSignOut, type RegisterBeforeSignOut } from "./pushSignOut.ts";

function hookRegistry() {
  const hooks = new Set<() => Promise<void>>();
  const register: RegisterBeforeSignOut = (hook) => {
    hooks.add(hook);
    return () => {
      hooks.delete(hook);
    };
  };
  return {
    register,
    hooks,
    async signOut() {
      await Promise.allSettled([...hooks].map((hook) => hook()));
    },
  };
}

test("signing out unregisters the device's token before the session goes away", async () => {
  const registry = hookRegistry();
  let token: string | null = "ExponentPushToken[abc]";
  const unregistered: string[] = [];

  unregisterOnSignOut(
    registry.register,
    () => token,
    async (t) => {
      unregistered.push(t);
    },
    () => {
      token = null;
    },
  );
  await registry.signOut();

  assert.deepEqual(unregistered, ["ExponentPushToken[abc]"]);
  assert.equal(token, null, "the registration is forgotten so a re-login registers again");
});

test("nothing is sent when this device never registered", async () => {
  const registry = hookRegistry();
  let calls = 0;
  unregisterOnSignOut(registry.register, () => null, async () => {
    calls++;
  }, () => {});
  await registry.signOut();
  assert.equal(calls, 0);
});

test("a failed unregister still forgets the token and surfaces the failure to the auth layer", async () => {
  const registry = hookRegistry();
  let token: string | null = "ExponentPushToken[abc]";
  unregisterOnSignOut(
    registry.register,
    () => token,
    async () => {
      throw new Error("offline");
    },
    () => {
      token = null;
    },
  );
  const [hook] = [...registry.hooks];
  await assert.rejects(hook!(), /offline/);
  assert.equal(token, null);
});

test("the returned cleanup removes the hook, so an unmounted app does not unregister", async () => {
  const registry = hookRegistry();
  let calls = 0;
  const off = unregisterOnSignOut(registry.register, () => "ExponentPushToken[abc]", async () => {
    calls++;
  }, () => {});
  off();
  await registry.signOut();
  assert.equal(calls, 0);
});
