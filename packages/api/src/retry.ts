import { Code, ConnectError } from "@connectrpc/connect";

/**
 * How many times a query may be retried before the error is surfaced.
 *
 * Two, as before. The change is which errors get there at all.
 */
export const MAX_QUERY_RETRIES = 2;

/**
 * Connect codes worth trying again.
 *
 * Everything else is a decision the server has already made and will make identically on the
 * next attempt. Retrying those costs three round trips before the user sees the message, and
 * on a phone that is three round trips of a spinner in place of "that email is already
 * registered".
 *
 * - `Unavailable` — the transport failed, or the service is not up. This is also the code
 *   connect-web produces for a network error, which is the case worth retrying on mobile:
 *   the tunnel a phone was on a second ago is gone and the next attempt uses a new one.
 * - `DeadlineExceeded` — one slow attempt says nothing about the next.
 * - `ResourceExhausted` — a server-side limiter is asking us to come back, which is the one
 *   4xx-shaped code that means "later", not "no".
 * - `Aborted` — a lost race on a contended row. Retrying is the documented response.
 * - `Unknown` — connect-web's code for a response it could not parse, which on React Native
 *   is usually a captive portal or a proxy interstitial rather than the service.
 */
const RETRYABLE = new Set<Code>([
  Code.Unavailable,
  Code.DeadlineExceeded,
  Code.ResourceExhausted,
  Code.Aborted,
  Code.Unknown,
]);

/**
 * Whether a failed query should be tried again.
 *
 * Note what is deliberately absent: `Unauthenticated` is not retried. The access token is
 * refreshed by @fm/auth before a request goes out, so a 401 that reaches here means the
 * refresh itself failed and the session is over — retrying it twice only delays the login
 * screen. `NotFound`, `InvalidArgument`, `AlreadyExists`, `PermissionDenied` and
 * `FailedPrecondition` are all answers, not failures.
 */
export function shouldRetryQuery(failureCount: number, error: unknown): boolean {
  if (failureCount >= MAX_QUERY_RETRIES) return false;

  // A non-Connect error is a bug in a hook or a serialisation failure, and repeating it
  // produces the same bug. Surface it.
  if (!(error instanceof ConnectError)) return false;

  return RETRYABLE.has(error.code);
}
