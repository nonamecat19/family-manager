import { Code, ConnectError } from "@connectrpc/connect";

export interface DisplayError {
  message: string;
  reference?: string;
  retryable: boolean;
}

const REFERENCE = /\(ref ([0-9a-z]{1,64})\)/i;

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

export function toDisplayError(error: unknown, fallback: string): DisplayError {
  if (!(error instanceof ConnectError)) {
    return { message: fallback, retryable: false };
  }

  const reference = REFERENCE.exec(error.rawMessage)?.[1];
  const retryable = RETRYABLE.has(error.code);

  if (error.code === Code.Internal || !HUMAN_MESSAGE.has(error.code)) {
    return { message: fallback, reference, retryable };
  }

  const message = error.rawMessage.trim();
  return { message: message === "" ? fallback : message, reference, retryable };
}

export function isRefreshRejection(error: unknown): boolean {
  if (!(error instanceof ConnectError)) return false;

  switch (error.code) {
    case Code.Unauthenticated:
    case Code.PermissionDenied:
    case Code.InvalidArgument:
    case Code.NotFound:
      return true;
    default:
      return false;
  }
}
