import assert from "node:assert/strict";
import { test } from "node:test";

import {
  DEFAULT_POLL_INTERVAL_S,
  SLOW_DOWN_STEP_S,
  hasExpired,
  isTerminalPhase,
  pollIntervalSeconds,
} from "./deviceLogin.ts";

test("only a decided or failed login stops the poll", () => {
  for (const phase of ["approved", "denied", "expired", "error"] as const) {
    assert.equal(isTerminalPhase(phase), true, phase);
  }
  for (const phase of ["idle", "starting", "pending"] as const) {
    assert.equal(isTerminalPhase(phase), false, phase);
  }
});

test("a pending reply keeps the interval the server offers", () => {
  assert.equal(pollIntervalSeconds(5, { slowDown: false, intervalSeconds: 7 }), 7);
  assert.equal(pollIntervalSeconds(5, { slowDown: false, intervalSeconds: 0 }), 5);
  assert.equal(pollIntervalSeconds(0, { slowDown: false }), DEFAULT_POLL_INTERVAL_S);
});

test("slow_down always backs off, even if the server offers less", () => {
  assert.equal(pollIntervalSeconds(5, { slowDown: true, intervalSeconds: 10 }), 10);
  assert.equal(pollIntervalSeconds(5, { slowDown: true, intervalSeconds: 0 }), 5 + SLOW_DOWN_STEP_S);
  assert.equal(pollIntervalSeconds(12, { slowDown: true, intervalSeconds: 10 }), 12 + SLOW_DOWN_STEP_S);
});

test("a grant without an expiry never expires on the client", () => {
  assert.equal(hasExpired(null, Date.now()), false);
  assert.equal(hasExpired(1_000, 999), false);
  assert.equal(hasExpired(1_000, 1_000), true);
});
