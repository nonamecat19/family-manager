import assert from "node:assert/strict";
import { test } from "node:test";

import { decodeAccessClaims } from "./claims.ts";

/** Builds a JWS-shaped token around a payload. The signature is never inspected. */
function token(payload: unknown, signature = "sig"): string {
  const b64url = (value: unknown) =>
    Buffer.from(JSON.stringify(value), "utf8")
      .toString("base64")
      .replace(/\+/g, "-")
      .replace(/\//g, "_")
      .replace(/=+$/, "");
  return `${b64url({ alg: "ES256", typ: "JWT" })}.${b64url(payload)}.${signature}`;
}

test("reads the three claims services/auth mints", () => {
  const claims = decodeAccessClaims(
    token({ sub: "user-1", family_id: "fam-1", email: "olena@example.com" }),
  );
  assert.deepEqual(claims, {
    userId: "user-1",
    familyId: "fam-1",
    email: "olena@example.com",
  });
});

test("a user in no family has an empty family id, not a null claims object", () => {
  // Normal on first launch — the token is minted without family_id until onboarding.
  const claims = decodeAccessClaims(token({ sub: "user-1", email: "o@example.com" }));
  assert.equal(claims?.userId, "user-1");
  assert.equal(claims?.familyId, "");
});

test("a token with no subject is unusable", () => {
  assert.equal(decodeAccessClaims(token({ family_id: "fam-1" })), null);
  assert.equal(decodeAccessClaims(token({ sub: "" })), null);
});

test("non-string claims are ignored rather than coerced", () => {
  const claims = decodeAccessClaims(token({ sub: "user-1", family_id: 42, email: null }));
  assert.equal(claims?.familyId, "");
  assert.equal(claims?.email, "");
});

test("multi-byte claims survive the base64url decode", () => {
  // atob yields one byte per character; without the UTF-8 step this comes back mojibake.
  const claims = decodeAccessClaims(token({ sub: "user-1", email: "олена@приклад.укр" }));
  assert.equal(claims?.email, "олена@приклад.укр");
});

test("base64url padding and alphabet are handled", () => {
  // A payload whose base64 length is not a multiple of four, containing bytes that encode to
  // the url-safe '-' and '_' characters.
  const claims = decodeAccessClaims(token({ sub: "a".repeat(7), email: "ÿÿÿ>?" }));
  assert.equal(claims?.userId, "aaaaaaa");
  assert.equal(claims?.email, "ÿÿÿ>?");
});

test("malformed input is null, never a throw", () => {
  for (const bad of [
    null,
    undefined,
    "",
    "not-a-token",
    "only.two",
    "a.b.c.d",
    "header..signature",
    `header.${Buffer.from("not json", "utf8").toString("base64url")}.sig`,
    `header.${Buffer.from("[1,2,3]", "utf8").toString("base64url")}.sig`,
    `header.${Buffer.from('"a string"', "utf8").toString("base64url")}.sig`,
  ]) {
    assert.equal(decodeAccessClaims(bad as string), null, String(bad));
  }
});

test("the signature is not inspected — this decodes, it does not verify", () => {
  // Documenting the contract: an unverified token parses fine here. Every service verifies
  // the signature itself, and that is the only judgement that grants access.
  const forged = token({ sub: "user-1", family_id: "fam-1" }, "obviously-not-a-signature");
  assert.equal(decodeAccessClaims(forged)?.userId, "user-1");
});
