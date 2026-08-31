import type { TranslationKey } from "../i18n/index.tsx";

/**
 * Calendar labels. Dates arrive as ISO `YYYY-MM-DD` strings — the shape `@fm/api`'s date
 * helpers and every finance RPC use — and come back as the exact strings the design draws.
 *
 * `Intl.DateTimeFormat` is not used, for the reason `money.ts` gives about `Intl.NumberFormat`,
 * plus one specific to Ukrainian: a day header needs the GENITIVE month ("29 серпня") while a
 * period header needs the nominative ("Серпень 2026"). One `Intl` call cannot give both, and
 * getting it wrong is the kind of mistake a Ukrainian reader notices immediately.
 */

/** The `t` from `useI18n()`. Passed in rather than hooked, so these stay pure functions. */
export type Translate = (key: TranslationKey, params?: Record<string, string | number>) => string;

/** `[year, month (1-12), day]` from an ISO date, with no timezone in the way. */
export function parseISO(iso: string): { year: number; month: number; day: number } {
  const [y = "1970", m = "01", d = "01"] = iso.slice(0, 10).split("-");
  return { year: Number(y), month: Number(m), day: Number(d) };
}

/** ISO weekday, 1 = Monday … 7 = Sunday — the numbering `common.weekdayShort.*` is keyed on. */
export function isoWeekday(iso: string): number {
  const { year, month, day } = parseISO(iso);
  const js = new Date(Date.UTC(year, month - 1, day)).getUTCDay();
  return js === 0 ? 7 : js;
}

function key(prefix: string, n: number): TranslationKey {
  return `${prefix}.${n}` as TranslationKey;
}

/** "Серпень" */
export function monthName(t: Translate, month: number): string {
  return t(key("common.month", month));
}

/** "серпень" — inside a sentence. */
export function monthNameLower(t: Translate, month: number): string {
  return t(key("common.monthLower", month));
}

/** "серпня" — dating a day. */
export function monthNameGenitive(t: Translate, month: number): string {
  return t(key("common.monthGen", month));
}

/** "Сер" — a chart axis. */
export function monthNameShort(t: Translate, month: number): string {
  return t(key("common.monthShort", month));
}

/** "Серпень 2026" — the period header's own label. */
export function monthTitle(t: Translate, year: number, month: number): string {
  return `${monthName(t, month)} ${year}`;
}

/** "29 серпня, сб" — the Transactions day heading. */
export function dayHeading(t: Translate, iso: string): string {
  const { month, day } = parseISO(iso);
  return t("common.dayHeading", {
    day,
    month: monthNameGenitive(t, month),
    weekday: t(key("common.weekdayShort", isoWeekday(iso))),
  });
}

/**
 * "сьогодні" / "вчора" / "2 дні" — the date strip in the add sheet. `today` is passed in
 * rather than read from the clock so a screen can render deterministically (and so a test,
 * were one written, would not depend on the day it ran).
 */
export function relativeDay(t: Translate, iso: string, today: string): string {
  const a = Date.UTC(parseISO(iso).year, parseISO(iso).month - 1, parseISO(iso).day);
  const b = Date.UTC(parseISO(today).year, parseISO(today).month - 1, parseISO(today).day);
  const days = Math.round((b - a) / 86_400_000);
  if (days === 0) return t("common.today");
  if (days === 1) return t("common.yesterday");
  return t("common.daysAgo", { count: days });
}

/** "8/30" — the compact day/month the add sheet's date strip puts above the word. */
export function shortDate(iso: string): string {
  const { month, day } = parseISO(iso);
  return `${month}/${day}`;
}
