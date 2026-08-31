import assert from "node:assert/strict";
import { test } from "node:test";

import {
  accountScope,
  currentPeriod,
  customPeriod,
  familyScope,
  memberScope,
  periodKey,
  periodWindow,
  scopeKey,
  stepPeriod,
} from "./scope.ts";

/* ------------------------------------------------------------------------- scope */

test("scopeKey is one comparable string per scope", () => {
  assert.equal(scopeKey(familyScope), "family");
  assert.equal(scopeKey(memberScope("m1")), "member:m1");
  assert.equal(scopeKey(accountScope("a1")), "account:a1");
  assert.equal(scopeKey(), "family", "no scope means the whole household");
});

test("scopes of different kinds never collide", () => {
  assert.notEqual(scopeKey(memberScope("x")), scopeKey(accountScope("x")));
});

/* ------------------------------------------------------------------------ window */

test("a month window is the whole calendar month, whatever the anchor", () => {
  assert.deepEqual(periodWindow({ granularity: "month", anchor: "2026-08-30" }), {
    from: "2026-08-01",
    to: "2026-08-31",
  });
  assert.deepEqual(periodWindow({ granularity: "month", anchor: "2024-02-15" }), {
    from: "2024-02-01",
    to: "2024-02-29",
  });
});

test("a week window honours the household's week start", () => {
  // 2026-08-30 is a Sunday.
  assert.deepEqual(periodWindow({ granularity: "week", anchor: "2026-08-30" }), {
    from: "2026-08-24",
    to: "2026-08-30",
  });
  assert.deepEqual(
    periodWindow({ granularity: "week", anchor: "2026-08-30" }, { weekStartsOn: 0 }),
    { from: "2026-08-30", to: "2026-09-05" },
  );
});

test("day and year windows", () => {
  assert.deepEqual(periodWindow({ granularity: "day", anchor: "2026-08-30" }), {
    from: "2026-08-30",
    to: "2026-08-30",
  });
  assert.deepEqual(periodWindow({ granularity: "year", anchor: "2026-08-30" }), {
    from: "2026-01-01",
    to: "2026-12-31",
  });
});

test("a custom range with a malformed edge fails here, not on the server", () => {
  assert.throws(() => periodWindow(customPeriod({ from: "2026-08-01", to: "not-a-date" })));
});

/* --------------------------------------------------------------------- period key */

test("two anchors inside one month are one cache entry", () => {
  assert.equal(
    periodKey({ granularity: "month", anchor: "2026-08-03" }),
    periodKey({ granularity: "month", anchor: "2026-08-27" }),
  );
});

test("adjacent windows never share a key", () => {
  assert.notEqual(
    periodKey({ granularity: "month", anchor: "2026-08-01" }),
    periodKey({ granularity: "month", anchor: "2026-09-01" }),
  );
  assert.notEqual(
    periodKey({ granularity: "day", anchor: "2026-08-30" }),
    periodKey({ granularity: "week", anchor: "2026-08-30" }),
  );
});

test("granularities of the same window are distinct keys", () => {
  // A single-day custom range and the day period cover the same dates but are different
  // requests; keying them alike would serve a day feed to a custom-range screen.
  assert.notEqual(
    periodKey({ granularity: "day", anchor: "2026-08-30" }),
    periodKey(customPeriod({ from: "2026-08-30", to: "2026-08-30" })),
  );
});

/* ---------------------------------------------------------------------- stepping */

test("stepping a month lands on the neighbouring month, not 30 days away", () => {
  assert.deepEqual(stepPeriod({ granularity: "month", anchor: "2026-08-30" }, 1), {
    granularity: "month",
    anchor: "2026-09-01",
  });
  assert.deepEqual(stepPeriod({ granularity: "month", anchor: "2026-01-31" }, 1), {
    granularity: "month",
    anchor: "2026-02-01",
  });
});

test("stepping is reversible for day, week and year", () => {
  for (const period of [
    { granularity: "day", anchor: "2026-03-01" },
    { granularity: "week", anchor: "2026-03-01" },
    { granularity: "year", anchor: "2026-01-01" },
  ] as const) {
    assert.deepEqual(stepPeriod(stepPeriod(period, 1), -1), period, period.granularity);
  }
});

test("currentPeriod defaults to the month around the given day", () => {
  assert.deepEqual(currentPeriod("month", new Date(2026, 7, 30)), {
    granularity: "month",
    anchor: "2026-08-30",
  });
});
