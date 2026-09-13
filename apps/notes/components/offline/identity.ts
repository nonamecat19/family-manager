const ALPHABET = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_";

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

export function ownerFileKey(ownerId: string): string {
  const safe = ownerId.replace(/[^A-Za-z0-9_-]/g, "").slice(0, 48);
  return `${safe}-${fingerprint(ownerId)}`;
}

function fingerprint(value: string): string {
  let hash = 5381;
  for (let i = 0; i < value.length; i += 1) {
    hash = ((hash << 5) + hash + value.charCodeAt(i)) | 0;
  }
  return (hash >>> 0).toString(36);
}

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
