import type { TranslationKey } from "../i18n/index.tsx";


export type Translate = (key: TranslationKey, params?: Record<string, string | number>) => string;

export function parseISO(iso: string): { year: number; month: number; day: number } {
  const [y = "1970", m = "01", d = "01"] = iso.slice(0, 10).split("-");
  return { year: Number(y), month: Number(m), day: Number(d) };
}

export function isoWeekday(iso: string): number {
  const { year, month, day } = parseISO(iso);
  const js = new Date(Date.UTC(year, month - 1, day)).getUTCDay();
  return js === 0 ? 7 : js;
}

function key(prefix: string, n: number): TranslationKey {
  return `${prefix}.${n}` as TranslationKey;
}

export function monthName(t: Translate, month: number): string {
  return t(key("common.month", month));
}

export function monthNameLower(t: Translate, month: number): string {
  return t(key("common.monthLower", month));
}

export function monthNameGenitive(t: Translate, month: number): string {
  return t(key("common.monthGen", month));
}

export function monthNameShort(t: Translate, month: number): string {
  return t(key("common.monthShort", month));
}

export function monthTitle(t: Translate, year: number, month: number): string {
  return `${monthName(t, month)} ${year}`;
}

export function dayHeading(t: Translate, iso: string): string {
  const { month, day } = parseISO(iso);
  return t("common.dayHeading", {
    day,
    month: monthNameGenitive(t, month),
    weekday: t(key("common.weekdayShort", isoWeekday(iso))),
  });
}

export function relativeDay(t: Translate, iso: string, today: string): string {
  const a = Date.UTC(parseISO(iso).year, parseISO(iso).month - 1, parseISO(iso).day);
  const b = Date.UTC(parseISO(today).year, parseISO(today).month - 1, parseISO(today).day);
  const days = Math.round((b - a) / 86_400_000);
  if (days === 0) return t("common.today");
  if (days === 1) return t("common.yesterday");
  return t("common.daysAgo", { count: days });
}

export function shortDate(iso: string): string {
  const { month, day } = parseISO(iso);
  return `${month}/${day}`;
}
