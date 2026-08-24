import assert from "node:assert/strict";
import { test } from "node:test";

import { Code, ConnectError } from "@connectrpc/connect";

import { toDisplayError } from "./errors.ts";

const FALLBACK = "Something went wrong.";

test("a message written for a person is shown as it is", () => {
  const err = new ConnectError("that email is already registered", Code.AlreadyExists);
  assert.equal(toDisplayError(err, FALLBACK).message, "that email is already registered");
});

// The service deliberately says nothing in an internal error; repeating it helps nobody.
test("an internal error shows the fallback and keeps the reference", () => {
  const err = new ConnectError("internal error (ref 3f1c8a02b7d5)", Code.Internal);
  const shown = toDisplayError(err, FALLBACK);

  assert.equal(shown.message, FALLBACK);
  assert.equal(shown.reference, "3f1c8a02b7d5");
});

test("an internal error without a reference still renders", () => {
  const shown = toDisplayError(new ConnectError("internal error", Code.Internal), FALLBACK);
  assert.equal(shown.message, FALLBACK);
  assert.equal(shown.reference, undefined);
});

test("transient failures are marked retryable", () => {
  assert.equal(toDisplayError(new ConnectError("x", Code.Unavailable), FALLBACK).retryable, true);
  assert.equal(
    toDisplayError(new ConnectError("x", Code.InvalidArgument), FALLBACK).retryable,
    false,
  );
});

test("a non-Connect throw becomes the fallback", () => {
  const shown = toDisplayError(new TypeError("x.map is not a function"), FALLBACK);
  assert.equal(shown.message, FALLBACK);
  assert.equal(shown.retryable, false);
  assert.equal(shown.reference, undefined);
});

// An empty message is a service bug, not a message; the user gets the fallback either way.
test("an empty message falls back", () => {
  assert.equal(toDisplayError(new ConnectError("", Code.NotFound), FALLBACK).message, FALLBACK);
});
