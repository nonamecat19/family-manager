/**
 * Reading the access token's own claims, for UI decisions only.
 *
 * SECURITY: this does NOT verify anything. The signature is not checked, the expiry is not
 * checked, and a token the device forged would parse here perfectly. Every service verifies
 * the token itself (ES256 over JWKS, `libs/go/auth`), and that is the only judgement that
 * decides what a request may do. What this is for is the other half — a screen that has to
 * decide whether to DRAW the "remove member" button, which needs to know who the caller is
 * before any round trip happens.
 *
 * Treating these claims as authorization would be a real vulnerability. Treating them as a
 * hint about what to render is exactly right, because the server refuses anything the hint
 * got wrong.
 *
 * The raw token still never leaves `SessionManager` — the provider hands out this derived
 * object, not the string it came from.
 */

export interface AccessClaims {
  /** The token subject: the signed-in user's id. */
  userId: string;
  /** The family the user belongs to; "" for a user who has not created or joined one. */
  familyId: string;
  /** Informational. Authorization never keys off it — see the note above. */
  email: string;
}

/**
 * Decodes the payload of a JWT. Returns null for anything that is not a readable JWT payload
 * rather than throwing: a malformed token is a "we do not know who this is" for rendering
 * purposes, and the next request will be rejected by the server anyway.
 */
export function decodeAccessClaims(accessToken: string | null | undefined): AccessClaims | null {
  const payload = decodePayload(accessToken);
  if (!payload) return null;

  // `sub` is the only claim a token is useless without; the other two are optional by design
  // (a user in no family has no family_id at all).
  const userId = typeof payload.sub === "string" ? payload.sub : "";
  if (userId === "") return null;

  return {
    userId,
    familyId: typeof payload["family_id"] === "string" ? payload["family_id"] : "",
    email: typeof payload["email"] === "string" ? payload["email"] : "",
  };
}

function decodePayload(token: string | null | undefined): Record<string, unknown> | null {
  if (!token) return null;
  const parts = token.split(".");
  // header.payload.signature — a JWS has exactly three segments.
  if (parts.length !== 3) return null;
  const segment = parts[1];
  if (!segment) return null;

  try {
    const json = utf8FromBase64Url(segment);
    const parsed: unknown = JSON.parse(json);
    if (typeof parsed !== "object" || parsed === null || Array.isArray(parsed)) return null;
    return parsed as Record<string, unknown>;
  } catch {
    return null;
  }
}

/**
 * base64url -> UTF-8 string.
 *
 * `atob` yields one byte per character, so a name or email outside ASCII would come back
 * mojibake if it were used directly. The bytes are percent-escaped and handed to `decodeURI`
 * component so multi-byte sequences survive — Hermes ships `atob` but no `TextDecoder`.
 */
function utf8FromBase64Url(segment: string): string {
  const base64 = segment.replace(/-/g, "+").replace(/_/g, "/");
  // `atob` requires the padding that base64url omits.
  const padded = base64.padEnd(base64.length + ((4 - (base64.length % 4)) % 4), "=");
  const binary = atob(padded);
  let escaped = "";
  for (let i = 0; i < binary.length; i++) {
    escaped += "%" + binary.charCodeAt(i).toString(16).padStart(2, "0");
  }
  return decodeURIComponent(escaped);
}
