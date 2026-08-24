import { Code, ConnectError } from "@connectrpc/connect";

/**
 * A failed call, reduced to what a screen needs.
 */
export interface DisplayError {
  /** The message to show. Never the raw wire text for an internal failure. */
  message: string;
  /**
   * The reference the services put in an opaque internal error, if there is one. Worth showing
   * next to the message: it is the only handle a user has on what went wrong, and the string
   * that finds the request in every service's logs (ADR 0008).
   */
  reference?: string;
  /** Whether trying the same thing again could plausibly work. */
  retryable: boolean;
}

/** Matches the tail of `internal error (ref 3f1c8a02b7d5)` that rpc.Internal produces. */
const REFERENCE = /\(ref ([0-9a-z]{1,64})\)/i;

/**
 * Codes whose message was written for a person and can be shown as-is.
 *
 * The services phrase these deliberately — "that email is already registered", "invitation
 * expired", "password must be at least 8 characters" — and replacing them with a generic
 * string would throw away the only part of the response the user can act on.
 */
const HUMAN_MESSAGE = new Set<Code>([
  Code.InvalidArgument,
  Code.AlreadyExists,
  Code.FailedPrecondition,
  Code.NotFound,
  Code.PermissionDenied,
  Code.Unauthenticated,
  Code.ResourceExhausted,
]);

const RETRYABLE = new Set<Code>([
  Code.Unavailable,
  Code.DeadlineExceeded,
  Code.Aborted,
  Code.Unknown,
  Code.ResourceExhausted,
]);

/**
 * Turns any thrown value into something a screen can render.
 *
 * `fallback` is the generic sentence to use when the error carries nothing worth showing —
 * supplied by the caller so it can be translated. Everything here is about which text to pick,
 * never about producing English.
 */
export function toDisplayError(error: unknown, fallback: string): DisplayError {
  if (!(error instanceof ConnectError)) {
    return { message: fallback, retryable: false };
  }

  const reference = REFERENCE.exec(error.rawMessage)?.[1];
  const retryable = RETRYABLE.has(error.code);

  // An internal error's text is deliberately opaque — "internal error (ref …)" — so showing it
  // tells the user nothing they can read. The reference travels separately.
  if (error.code === Code.Internal || !HUMAN_MESSAGE.has(error.code)) {
    return { message: fallback, reference, retryable };
  }

  const message = error.rawMessage.trim();
  return { message: message === "" ? fallback : message, reference, retryable };
}
