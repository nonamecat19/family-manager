import assert from "node:assert/strict";
import { test } from "node:test";

import {
  addDays,
  addMonths,
  contains,
  dayCount,
  endOfMonth,
  fromISODate,
  isISODate,
  periodRange,
  shiftPeriod,
  startOfMonth,
  toISODate,
} from "./dates.ts";

// A Thursday, deliberately near a month edge.
const REFERENCE = new Date(2026, 2, 12); // 2026-03-12

test("toISODate uses the local calendar day", () => {
  assert.equal(toISODate(new Date(2026, 0, 5)), "2026-01-05");
  // 23:30 local must still be that local day, not tomorrow in UTC.
  assert.equal(toISODate(new Date(2026, 0, 31, 23, 30)), "2026-01-31");
});

test("fromISODate rejects malformed and impossible dates", () => {
  assert.equal(toISODate(fromISODate("2026-03-12")), "2026-03-12");
  for (const bad of ["2026-3-12", "12-03-2026", "2026-02-30", "not-a-date", ""]) {
    assert.throws(() => fromISODate(bad), `expected a throw for ${bad}`);
    assert.equal(isISODate(bad), false);
  }
});

test("addDays crosses month and year boundaries", () => {
  assert.equal(addDays("2026-01-31", 1), "2026-02-01");
  assert.equal(addDays("2026-12-31", 1), "2027-01-01");
  assert.equal(addDays("2026-03-01", -1), "2026-02-28");
  assert.equal(addDays("2028-03-01", -1), "2028-02-29"); // leap year
});

test("addMonths clamps instead of overflowing", () => {
  assert.equal(addMonths("2026-01-31", 1), "2026-02-28");
  assert.equal(addMonths("2028-01-31", 1), "2028-02-29");
  assert.equal(addMonths("2026-03-31", -1), "2026-02-28");
  assert.equal(addMonths("2026-12-15", 1), "2027-01-15");
});

test("periodRange: day", () => {
  assert.deepEqual(periodRange("day", { today: REFERENCE }), {
    from: "2026-03-12",
    to: "2026-03-12",
  });
});

test("periodRange: week honours the week start", () => {
  assert.deepEqual(periodRange("week", { today: REFERENCE, weekStartsOn: 1 }), {
    from: "2026-03-09",
    to: "2026-03-15",
  });
  assert.deepEqual(periodRange("week", { today: REFERENCE, weekStartsOn: 0 }), {
    from: "2026-03-08",
    to: "2026-03-14",
  });
});

test("periodRange: month covers the whole calendar month", () => {
  assert.deepEqual(periodRange("month", { today: REFERENCE }), {
    from: "2026-03-01",
    to: "2026-03-31",
  });
  assert.deepEqual(periodRange("month", { today: new Date(2026, 1, 5) }), {
    from: "2026-02-01",
    to: "2026-02-28",
  });
});

test("periodRange: year and all", () => {
  assert.deepEqual(periodRange("year", { today: REFERENCE }), {
    from: "2026-01-01",
    to: "2026-12-31",
  });
  assert.deepEqual(periodRange("all", { today: REFERENCE }), {
    from: "1970-01-01",
    to: "2026-03-12",
  });
});

test("shiftPeriod steps whole periods", () => {
  const march = periodRange("month", { today: REFERENCE });
  assert.deepEqual(shiftPeriod(march, "month", -1), { from: "2026-02-01", to: "2026-02-28" });
  assert.deepEqual(shiftPeriod(march, "month", 1), { from: "2026-04-01", to: "2026-04-30" });

  const week = periodRange("week", { today: REFERENCE });
  assert.deepEqual(shiftPeriod(week, "week", 1), { from: "2026-03-16", to: "2026-03-22" });

  assert.deepEqual(shiftPeriod({ from: "2026-01-01", to: "2026-12-31" }, "year", -1), {
    from: "2025-01-01",
    to: "2025-12-31",
  });
});

test("shifting a 31-day month back and forth returns the original month", () => {
  const may = { from: "2026-05-01", to: "2026-05-31" };
  const backAndForth = shiftPeriod(shiftPeriod(may, "month", -1), "month", 1);
  assert.deepEqual(backAndForth, may);
});

test("startOfMonth and endOfMonth", () => {
  assert.equal(startOfMonth("2026-03-12"), "2026-03-01");
  assert.equal(endOfMonth("2026-02-01"), "2026-02-28");
  assert.equal(endOfMonth("2028-02-10"), "2028-02-29");
});

test("contains is inclusive on both ends", () => {
  const range = { from: "2026-03-01", to: "2026-03-31" };
  assert.ok(contains(range, "2026-03-01"));
  assert.ok(contains(range, "2026-03-31"));
  assert.ok(!contains(range, "2026-02-28"));
  assert.ok(!contains(range, "2026-04-01"));
});

test("dayCount is inclusive", () => {
  assert.equal(dayCount({ from: "2026-03-01", to: "2026-03-01" }), 1);
  assert.equal(dayCount({ from: "2026-03-01", to: "2026-03-31" }), 31);
  assert.equal(dayCount({ from: "2026-01-01", to: "2026-12-31" }), 365);
});

test("dayCount is unaffected by a DST transition", () => {
  // Europe/Kyiv springs forward on 2026-03-29; the day count must stay 31.
  assert.equal(dayCount({ from: "2026-03-01", to: "2026-03-31" }), 31);
});
