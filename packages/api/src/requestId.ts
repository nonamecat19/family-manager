import type { Interceptor } from "@connectrpc/connect";

/**
 * The header services read and echo. Matches `rpc.RequestIDHeader` in libs/go/rpc — an id sent
 * under any other name is simply ignored and the service mints its own.
 */
export const REQUEST_ID_HEADER = "X-Request-Id";

/**
 * Twelve hex characters. Long enough that a household's traffic will not collide, short enough
 * to read aloud, and well inside the 64-byte printable-ASCII bound the services enforce on an
 * id they did not mint.
 */
const ID_BYTES = 6;

/**
 * Generates a request id.
 *
 * `crypto.getRandomValues` is present in Hermes and on web; the arithmetic fallback exists
 * because a correlation id is not a security value and an app that cannot produce one should
 * still be traceable rather than silently anonymous.
 */
export function newRequestId(): string {
  const bytes = new Uint8Array(ID_BYTES);
  const webcrypto = globalThis.crypto;
  if (webcrypto?.getRandomValues) {
    webcrypto.getRandomValues(bytes);
  } else {
    for (let i = 0; i < bytes.length; i++) bytes[i] = Math.floor(Math.random() * 256);
  }
  return Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
}

/**
 * Stamps every outgoing call with a request id.
 *
 * The services accept the caller's id and use it as the reference in an opaque internal error
 * (ADR 0008), so the string a user reads off a failure screen is the string that finds the
 * request in the service logs. Without this the id is minted server-side and the app never
 * learns it, which makes "it failed at about four o'clock" the whole bug report.
 *
 * An id already set by a caller is left alone: a retry should not become a second trace of
 * what the user experienced as one action.
 */
export const requestIdInterceptor: Interceptor = (next) => async (req) => {
  if (!req.header.get(REQUEST_ID_HEADER)) {
    req.header.set(REQUEST_ID_HEADER, newRequestId());
  }
  return next(req);
};
