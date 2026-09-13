import type { Interceptor } from "@connectrpc/connect";

export const REQUEST_ID_HEADER = "X-Request-Id";

const ID_BYTES = 6;

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

export const requestIdInterceptor: Interceptor = (next) => async (req) => {
  if (!req.header.get(REQUEST_ID_HEADER)) {
    req.header.set(REQUEST_ID_HEADER, newRequestId());
  }
  return next(req);
};
