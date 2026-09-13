import { Code, ConnectError } from "@connectrpc/connect";

export const MAX_QUERY_RETRIES = 2;

const RETRYABLE = new Set<Code>([
  Code.Unavailable,
  Code.DeadlineExceeded,
  Code.ResourceExhausted,
  Code.Aborted,
  Code.Unknown,
]);

export function shouldRetryQuery(failureCount: number, error: unknown): boolean {
  if (failureCount >= MAX_QUERY_RETRIES) return false;

  if (!(error instanceof ConnectError)) return false;

  return RETRYABLE.has(error.code);
}
