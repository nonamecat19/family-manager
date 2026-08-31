/**
 * Scope and Period — the two controls every finance screen shares.
 *
 * The Home switcher ("Родина ▾" → family / one member / one account) and the period tabs
 * (день/тиждень/місяць/рік/період) are the same two values on Home, Transactions, Charts and
 * every widget. They are modelled once, here, for three reasons:
 *
 *  - a cache key must be *stable*: two anchors inside the same month are the same window, so
 *    they must produce the same key or every month-stepper tap refetches what is already cached;
 *  - the window edges must agree with the server's, so `periodWindow` is the client-side twin
 *    of the resolver behind `finance.v1.Period`;
 *  - screens speak a small domain union ({ kind: "member", memberId }), never the wire enums.
 *
 * This module is deliberately free of any `@fm/sdk` import. Query keys are a cache concern,
 * not a wire concern, and keeping the dependency out is also what lets `node --test` load this
 * file and queryKeys.ts: the generated SDK declares TypeScript `enum`s, which are not erasable
 * syntax and so cannot be type-stripped. The wire mapping lives in scopeWire.ts.
 */

import { addDays, addMonths, endOfMonth, fromISODate, startOfMonth, toISODate, type DateRange } from "./dates.ts";

/* ------------------------------------------------------------------------ scope */

export type ScopeInput =
  | { kind: "family" }
  | { kind: "member"; memberId: string }
  | { kind: "account"; accountId: string };

/** The default scope: the whole household. */
export const familyScope: ScopeInput = { kind: "family" };

export function memberScope(memberId: string): ScopeInput {
  return { kind: "member", memberId };
}

export function accountScope(accountId: string): ScopeInput {
  return { kind: "account", accountId };
}

/**
 * A single string per scope, so a query key is comparable by value. `family` carries no id:
 * a family scope with a stale memberId lying around must not miss the cache entry.
 */
export function scopeKey(scope: ScopeInput = familyScope): string {
  switch (scope.kind) {
    case "member":
      return `member:${scope.memberId}`;
    case "account":
      return `account:${scope.accountId}`;
    case "family":
      return "family";
  }
}

/* ----------------------------------------------------------------------- period */

export type PeriodGranularityInput = "day" | "week" | "month" | "year";

export type PeriodInput =
  | { granularity: PeriodGranularityInput; anchor: string }
  | { granularity: "custom"; range: DateRange };

/** A steppable period — the arrows either side of "Серпень 2026". Custom ranges have none. */
export type SteppablePeriod = Extract<PeriodInput, { anchor: string }>;

export interface PeriodWindowOptions {
  /** 0 = Sunday, 1 = Monday (the default, and the household's setting on screen 11). */
  weekStartsOn?: 0 | 1;
}

export function dayPeriod(anchor: string): PeriodInput {
  return { granularity: "day", anchor };
}

export function monthPeriod(anchor: string): PeriodInput {
  return { granularity: "month", anchor };
}

export function customPeriod(range: DateRange): PeriodInput {
  return { granularity: "custom", range };
}

/** The period covering "today", for a screen opening with no user choice yet. */
export function currentPeriod(
  granularity: PeriodGranularityInput = "month",
  today: Date = new Date(),
): PeriodInput {
  return { granularity, anchor: toISODate(today) };
}

/**
 * The inclusive window a period resolves to. Anchored on any day inside the window, which is
 * what lets the month stepper carry a single date around instead of a pair.
 */
export function periodWindow(period: PeriodInput, opts: PeriodWindowOptions = {}): DateRange {
  if (period.granularity === "custom") {
    // Validate both ends: a malformed custom range must fail here, not silently key a cache
    // entry the server will reject.
    fromISODate(period.range.from);
    fromISODate(period.range.to);
    return period.range;
  }

  const anchor = period.anchor;
  const date = fromISODate(anchor);

  switch (period.granularity) {
    case "day":
      return { from: anchor, to: anchor };
    case "week": {
      const weekStartsOn = opts.weekStartsOn ?? 1;
      const offset = (date.getDay() - weekStartsOn + 7) % 7;
      const from = addDays(anchor, -offset);
      return { from, to: addDays(from, 6) };
    }
    case "month":
      return { from: startOfMonth(anchor), to: endOfMonth(anchor) };
    case "year": {
      const year = anchor.slice(0, 4);
      return { from: `${year}-01-01`, to: `${year}-12-31` };
    }
  }
}

/**
 * One string per *window*, not per anchor: the 3rd and the 27th of August are one cache entry.
 * Without this every day the user opens the app would start a cold month.
 */
export function periodKey(period: PeriodInput, opts: PeriodWindowOptions = {}): string {
  const window = periodWindow(period, opts);
  switch (period.granularity) {
    case "day":
      return `day:${window.from}`;
    case "week":
      return `week:${window.from}`;
    case "month":
      return `month:${window.from.slice(0, 7)}`;
    case "year":
      return `year:${window.from.slice(0, 4)}`;
    case "custom":
      return `custom:${window.from}..${window.to}`;
  }
}

/** Steps a period by whole windows — the ◀ ▶ either side of the period label. */
export function stepPeriod(period: SteppablePeriod, by: number): SteppablePeriod {
  switch (period.granularity) {
    case "day":
      return { granularity: "day", anchor: addDays(period.anchor, by) };
    case "week":
      return { granularity: "week", anchor: addDays(period.anchor, by * 7) };
    case "month":
      return { granularity: "month", anchor: addMonths(startOfMonth(period.anchor), by) };
    case "year":
      return {
        granularity: "year",
        anchor: `${Number(period.anchor.slice(0, 4)) + by}-01-01`,
      };
  }
}
