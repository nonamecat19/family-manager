export interface AccessClaims {
  userId: string;
  familyId: string;
  email: string;
}

export function decodeAccessClaims(accessToken: string | null | undefined): AccessClaims | null {
  const payload = decodePayload(accessToken);
  if (!payload) return null;

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

function utf8FromBase64Url(segment: string): string {
  const base64 = segment.replace(/-/g, "+").replace(/_/g, "/");
  const padded = base64.padEnd(base64.length + ((4 - (base64.length % 4)) % 4), "=");
  const binary = atob(padded);
  let escaped = "";
  for (let i = 0; i < binary.length; i++) {
    escaped += "%" + binary.charCodeAt(i).toString(16).padStart(2, "0");
  }
  return decodeURIComponent(escaped);
}
