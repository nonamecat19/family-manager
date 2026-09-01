import assert from "node:assert/strict";
import test from "node:test";

import { ownerFileKey, userIdFromToken } from "./identity.ts";

/**
 * Reading the queue's owner out of an access token. The queue file's name and its stamp both
 * come from here, so "which user is this" has to survive a token that is missing, malformed, or
 * carries claims this app did not expect.
 */

function token(claims: Record<string, unknown>): string {
  const payload = Buffer.from(JSON.stringify(claims), "utf8")
    .toString("base64")
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/, "");
  return `eyJhbGciOiJFUzI1NiJ9.${payload}.c2lnbmF0dXJl`;
}

test("the owner is the token's subject", () => {
  assert.equal(
    userIdFromToken(token({ sub: "0193f2b1-0000-7000-8000-000000000001", family_id: "fam" })),
    "0193f2b1-0000-7000-8000-000000000001",
  );
});

test("a non-ASCII claim does not corrupt the subject", () => {
  // The payload is UTF-8, and a display name with an accent in it used to be enough to make a
  // naive latin1 decode throw or return mojibake — which would have meant no owner, and a queue
  // that quietly stops persisting.
  assert.equal(userIdFromToken(token({ sub: "user-1", name: "Renée Ø 家" })), "user-1");
});

test("anything that is not a token with a subject has no owner", () => {
  assert.equal(userIdFromToken(null), null);
  assert.equal(userIdFromToken(undefined), null);
  assert.equal(userIdFromToken(""), null);
  assert.equal(userIdFromToken("not-a-jwt"), null);
  assert.equal(userIdFromToken("a.!!!not-base64!!!.c"), null);
  assert.equal(userIdFromToken(token({ family_id: "fam" })), null);
  assert.equal(userIdFromToken(token({ sub: "" })), null);
  assert.equal(userIdFromToken(token({ sub: 12 })), null);
});

test("two owners never share a file key, and the key is a safe file name", () => {
  const a = ownerFileKey("0193f2b1-0000-7000-8000-000000000001");
  const b = ownerFileKey("0193f2b1-0000-7000-8000-000000000002");

  assert.notEqual(a, b);
  assert.match(a, /^[A-Za-z0-9_-]+$/);
  // Characters a file name has no business carrying are dropped, but the id still decides the
  // key: two ids that differ only in a stripped character must not collapse onto one file.
  assert.notEqual(ownerFileKey("../../etc/passwd"), ownerFileKey("etcpasswd"));
  assert.match(ownerFileKey("../../etc/passwd"), /^[A-Za-z0-9_-]+$/);
});
