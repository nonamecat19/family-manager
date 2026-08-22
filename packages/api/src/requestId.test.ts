import assert from "node:assert/strict";
import { test } from "node:test";

import { REQUEST_ID_HEADER, newRequestId, requestIdInterceptor } from "./requestId.ts";

test("ids are hex, short, and distinct", () => {
  const seen = new Set<string>();
  for (let i = 0; i < 500; i++) {
    const id = newRequestId();
    assert.match(id, /^[0-9a-f]{12}$/, `unexpected shape: ${id}`);
    assert.equal(seen.has(id), false, `repeated id ${id}`);
    seen.add(id);
  }
});

// Services bound an incoming id at 64 bytes of printable ASCII and drop anything else, so an id
// that fails those rules is silently replaced and correlation is lost without an error.
test("ids satisfy the bound services enforce", () => {
  const id = newRequestId();
  assert.ok(id.length <= 64);
  assert.match(id, /^[\x21-\x7e]+$/);
});

/** Minimal stand-in for a Connect request: the interceptor only touches headers. */
function fakeRequest(headers: Record<string, string> = {}) {
  return { header: new Headers(headers) };
}

test("the interceptor stamps an id on an unmarked request", async () => {
  const req = fakeRequest();
  let seen: string | null = null;

  await requestIdInterceptor(
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (async (r: any) => {
      seen = r.header.get(REQUEST_ID_HEADER);
      return {} as never;
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
    }) as any,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
  )(req as any);

  assert.match(seen ?? "", /^[0-9a-f]{12}$/);
});

// A retry is the same action from the user's point of view, so it keeps the id it already had.
test("an id already present is left alone", async () => {
  const req = fakeRequest({ [REQUEST_ID_HEADER]: "abc123" });
  let seen: string | null = null;

  await requestIdInterceptor(
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (async (r: any) => {
      seen = r.header.get(REQUEST_ID_HEADER);
      return {} as never;
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
    }) as any,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
  )(req as any);

  assert.equal(seen, "abc123");
});
