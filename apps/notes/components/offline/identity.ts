/**
 * Who the capture queue belongs to.
 *
 * The queue is written to the device's document directory, which outlives a sign-out and is
 * shared by everyone who uses the tablet. So every queued note has to be attributable, and the
 * only identity this app holds is the access token — @fm/auth deliberately exposes tokens and a
 * status, never a user. The subject claim is read out of the token here.
 *
 * READING A JWT IS NOT VERIFYING ONE. Nothing in this file authorizes anything: the id is used
 * as a local storage key and as a stamp the queue checks its own file against, and every actual
 * permission decision still happens server-side against the signed token. A forged token would
 * buy an attacker a queue file named after somebody — and no note in it would ever be created,
 * because the server rejects the signature. Verifying ES256 on the client would need a JWKS
 * fetch and a crypto dependency to answer a question the client is not the one asking.
 *
 * The claim is `sub`, stamped by services/auth and read back by libs/go/auth (claims.go:72).
 */

/** Base64url, plus the two standard-alphabet characters normalised into it. */
const ALPHABET = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_";

/**
 * The user id inside an access token, or null for anything that is not a token with a subject.
 * Every failure is a null rather than a throw: a queue with no owner simply does not persist.
 */
export function userIdFromToken(token: string | null | undefined): string | null {
  if (!token) return null;
  const payload = token.split(".")[1];
  if (payload === undefined || payload === "") return null;
  try {
    const claims: unknown = JSON.parse(utf8FromBase64Url(payload));
    if (typeof claims !== "object" || claims === null) return null;
    const sub = (claims as { sub?: unknown }).sub;
    return typeof sub === "string" && sub !== "" ? sub : null;
  } catch {
    return null;
  }
}

/**
 * The file name segment for an owner: safe characters from the id, plus a hash of the whole id
 * so two ids can never collapse onto one name. A collision would still not leak anything — the
 * file names its owner and hydration refuses a file stamped with anyone else — it would only
 * cost the loser their queue, so the hash is about correctness, not secrecy.
 */
export function ownerFileKey(ownerId: string): string {
  const safe = ownerId.replace(/[^A-Za-z0-9_-]/g, "").slice(0, 48);
  return `${safe}-${fingerprint(ownerId)}`;
}

/** djb2, base36. Not a security hash: it names a file. */
function fingerprint(value: string): string {
  let hash = 5381;
  for (let i = 0; i < value.length; i += 1) {
    hash = ((hash << 5) + hash + value.charCodeAt(i)) | 0;
  }
  return (hash >>> 0).toString(36);
}

/**
 * Decodes a base64url segment as UTF-8.
 *
 * Hand-rolled because Hermes has no `atob` (apps/recipes decodes base64 by hand for the same
 * reason), and the bytes become a string through decodeURIComponent's percent form so a claim
 * with a non-ASCII character — a display name in the token — does not come back mojibake.
 */
function utf8FromBase64Url(segment: string): string {
  const normalised = segment.replace(/\+/g, "-").replace(/\//g, "_");
  let buffer = 0;
  let bits = 0;
  let percent = "";
  for (const char of normalised) {
    if (char === "=") break;
    const value = ALPHABET.indexOf(char);
    if (value < 0) throw new Error("not base64url");
    buffer = (buffer << 6) | value;
    bits += 6;
    if (bits >= 8) {
      bits -= 8;
      percent += `%${((buffer >> bits) & 0xff).toString(16).padStart(2, "0")}`;
    }
  }
  return decodeURIComponent(percent);
}
