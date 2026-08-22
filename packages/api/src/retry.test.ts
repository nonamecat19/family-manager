import assert from "node:assert/strict";
import { test } from "node:test";

import { Code, ConnectError } from "@connectrpc/connect";

import { MAX_QUERY_RETRIES, shouldRetryQuery } from "./retry.ts";

const err = (code: Code) => new ConnectError("boom", code);

test("transient failures are retried", () => {
  for (const code of [
    Code.Unavailable,
    Code.DeadlineExceeded,
    Code.ResourceExhausted,
    Code.Aborted,
    Code.Unknown,
  ]) {
    assert.equal(shouldRetryQuery(0, err(code)), true, `${Code[code]} should retry`);
  }
});

test("answers the server already gave are not retried", () => {
  for (const code of [
    Code.InvalidArgument,
    Code.NotFound,
    Code.AlreadyExists,
    Code.PermissionDenied,
    Code.FailedPrecondition,
    Code.Unimplemented,
  ]) {
    assert.equal(shouldRetryQuery(0, err(code)), false, `${Code[code]} should not retry`);
  }
});

// The token is refreshed before the request goes out, so a 401 arriving here means the refresh
// failed. Two more attempts only delay the login screen.
test("unauthenticated is not retried", () => {
  assert.equal(shouldRetryQuery(0, err(Code.Unauthenticated)), false);
});

test("retries stop at the limit", () => {
  assert.equal(shouldRetryQuery(MAX_QUERY_RETRIES - 1, err(Code.Unavailable)), true);
  assert.equal(shouldRetryQuery(MAX_QUERY_RETRIES, err(Code.Unavailable)), false);
});

// Anything that is not a ConnectError got here from our own code, and repeating it repeats the
// bug rather than working around a network.
test("non-Connect errors are surfaced immediately", () => {
  assert.equal(shouldRetryQuery(0, new TypeError("x.map is not a function")), false);
  assert.equal(shouldRetryQuery(0, "not even an error"), false);
});
