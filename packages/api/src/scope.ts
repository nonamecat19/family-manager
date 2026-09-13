import { addDays, addMonths, endOfMonth, fromISODate, startOfMonth, toISODate, type DateRange } from "./dates.ts";


export type ScopeInput =
  | { kind: "family" }
  | { kind: "member"; memberId: string }
  | { kind: "account"; accountId: string };

export const familyScope: ScopeInput = { kind: "family" };

export function memberScope(memberId: string): ScopeInput {
  return { kind: "member", memberId };
}

export function accountScope(accountId: string): ScopeInput {
  return { kind: "account", accountId };
}

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


export type PeriodGranularityInput = "day" | "week" | "month" | "year";

export type PeriodInput =
  | { granularity: PeriodGranularityInput; anchor: string }
  | { granularity: "custom"; range: DateRange };

export type SteppablePeriod = Extract<PeriodInput, { anchor: string }>;

export interface PeriodWindowOptions {
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

export function currentPeriod(
  granularity: PeriodGranularityInput = "month",
  today: Date = new Date(),
): PeriodInput {
  return { granularity, anchor: toISODate(today) };
}

export function periodWindow(period: PeriodInput, opts: PeriodWindowOptions = {}): DateRange {
  if (period.granularity === "custom") {
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
