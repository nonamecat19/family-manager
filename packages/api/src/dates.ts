/**
 * Calendar helpers for reports. A spending period is a calendar question ("this month"), not
 * an instant one, so dates are handled as `YYYY-MM-DD` strings in the user's local calendar
 * and never as UTC timestamps — the classic bug is a transaction added at 23:30 on the 31st
 * landing in next month's report.
 */

export interface DateRange {
  /** Inclusive start, YYYY-MM-DD. */
  from: string;
  /** Inclusive end, YYYY-MM-DD. */
  to: string;
}

export type PeriodKind = "day" | "week" | "month" | "year" | "all" | "custom";

const ISO_DATE = /^\d{4}-\d{2}-\d{2}$/;

/** Formats a Date's *local* calendar day. */
export function toISODate(d: Date): string {
  const y = d.getFullYear();
  const m = `${d.getMonth() + 1}`.padStart(2, "0");
  const day = `${d.getDate()}`.padStart(2, "0");
  return `${y}-${m}-${day}`;
}

/** Parses YYYY-MM-DD into a local-midnight Date. Throws on anything else. */
export function fromISODate(s: string): Date {
  if (!ISO_DATE.test(s)) throw new Error(`dates: ${JSON.stringify(s)} is not YYYY-MM-DD`);
  const [y, m, d] = s.split("-").map(Number) as [number, number, number];
  const date = new Date(y, m - 1, d);
  if (date.getFullYear() !== y || date.getMonth() !== m - 1 || date.getDate() !== d) {
    throw new Error(`dates: ${s} is not a real calendar date`);
  }
  return date;
}

export function isISODate(s: string): boolean {
  if (!ISO_DATE.test(s)) return false;
  try {
    fromISODate(s);
    return true;
  } catch {
    return false;
  }
}

export function addDays(s: string, days: number): string {
  const d = fromISODate(s);
  d.setDate(d.getDate() + days);
  return toISODate(d);
}

export function addMonths(s: string, months: number): string {
  const d = fromISODate(s);
  const targetMonth = d.getMonth() + months;
  const anchor = new Date(d.getFullYear(), targetMonth, 1);
  // Clamp: one month after Jan 31 is Feb 28/29, not Mar 3.
  const lastDay = daysInMonth(anchor.getFullYear(), anchor.getMonth());
  anchor.setDate(Math.min(d.getDate(), lastDay));
  return toISODate(anchor);
}

function daysInMonth(year: number, monthIndex: number): number {
  return new Date(year, monthIndex + 1, 0).getDate();
}

export interface PeriodOptions {
  /** 0 = Sunday, 1 = Monday (the default — most of the world, and the app's origin locale). */
  weekStartsOn?: 0 | 1;
  /** The day treated as "today"; defaults to the system date. Injected by tests. */
  today?: Date;
}

/** The range covering `kind` around a reference day. "all" starts at the Unix epoch. */
export function periodRange(kind: Exclude<PeriodKind, "custom">, opts: PeriodOptions = {}): DateRange {
  const today = opts.today ?? new Date();
  const weekStartsOn = opts.weekStartsOn ?? 1;

  switch (kind) {
    case "day": {
      const d = toISODate(today);
      return { from: d, to: d };
    }
    case "week": {
      const offset = (today.getDay() - weekStartsOn + 7) % 7;
      const start = new Date(today.getFullYear(), today.getMonth(), today.getDate() - offset);
      return { from: toISODate(start), to: addDays(toISODate(start), 6) };
    }
    case "month": {
      const start = new Date(today.getFullYear(), today.getMonth(), 1);
      const end = new Date(today.getFullYear(), today.getMonth() + 1, 0);
      return { from: toISODate(start), to: toISODate(end) };
    }
    case "year": {
      return { from: `${today.getFullYear()}-01-01`, to: `${today.getFullYear()}-12-31` };
    }
    case "all": {
      return { from: "1970-01-01", to: toISODate(today) };
    }
  }
}

/** Shifts a range by whole periods — the arrows either side of the period label. */
export function shiftPeriod(
  range: DateRange,
  kind: Exclude<PeriodKind, "custom" | "all">,
  by: number,
): DateRange {
  switch (kind) {
    case "day":
      return { from: addDays(range.from, by), to: addDays(range.to, by) };
    case "week":
      return { from: addDays(range.from, by * 7), to: addDays(range.to, by * 7) };
    case "month": {
      const from = addMonths(startOfMonth(range.from), by);
      return { from, to: endOfMonth(from) };
    }
    case "year": {
      const year = Number(range.from.slice(0, 4)) + by;
      return { from: `${year}-01-01`, to: `${year}-12-31` };
    }
  }
}

export function startOfMonth(s: string): string {
  return `${s.slice(0, 7)}-01`;
}

export function endOfMonth(s: string): string {
  const d = fromISODate(s);
  return toISODate(new Date(d.getFullYear(), d.getMonth() + 1, 0));
}

export function contains(range: DateRange, day: string): boolean {
  return day >= range.from && day <= range.to;
}

/** Inclusive day count — a range's denominator when averaging spend per day. */
export function dayCount(range: DateRange): number {
  const ms = fromISODate(range.to).getTime() - fromISODate(range.from).getTime();
  return Math.round(ms / 86_400_000) + 1;
}

/** Label for the period header: "March 2026", "1–7 Mar 2026", "12 Mar 2026". */
export function formatRange(range: DateRange, locale?: string): string {
  const from = fromISODate(range.from);
  const to = fromISODate(range.to);

  if (range.from === range.to) {
    return from.toLocaleDateString(locale, { day: "numeric", month: "short", year: "numeric" });
  }
  if (range.from === startOfMonth(range.from) && range.to === endOfMonth(range.to)) {
    if (from.getMonth() === to.getMonth() && from.getFullYear() === to.getFullYear()) {
      return from.toLocaleDateString(locale, { month: "long", year: "numeric" });
    }
  }
  if (range.from === `${from.getFullYear()}-01-01` && range.to === `${to.getFullYear()}-12-31`) {
    return `${from.getFullYear()}`;
  }

  const fromLabel = from.toLocaleDateString(locale, { day: "numeric", month: "short" });
  const toLabel = to.toLocaleDateString(locale, { day: "numeric", month: "short", year: "numeric" });
  return `${fromLabel} – ${toLabel}`;
}
