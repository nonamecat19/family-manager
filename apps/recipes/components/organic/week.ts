import type { TranslationKey } from "../i18n/index.tsx";

export function toISODate(d: Date): string {
  const year = d.getFullYear();
  const month = `${d.getMonth() + 1}`.padStart(2, "0");
  const day = `${d.getDate()}`.padStart(2, "0");
  return `${year}-${month}-${day}`;
}

export function weekRange(date: Date): { from: string; to: string } {
  const start = new Date(date);
  start.setDate(date.getDate() - date.getDay());
  const end = new Date(start);
  end.setDate(start.getDate() + 6);
  return { from: toISODate(start), to: toISODate(end) };
}

export interface WeekDay {
  iso: string;
  dayKey: TranslationKey;
  num: number;
  isToday: boolean;
}

const DAY_KEYS: TranslationKey[] = [
  "weekdays.sun",
  "weekdays.mon",
  "weekdays.tue",
  "weekdays.wed",
  "weekdays.thu",
  "weekdays.fri",
  "weekdays.sat",
];

export function buildWeek(date: Date): WeekDay[] {
  const start = new Date(date);
  start.setDate(date.getDate() - date.getDay());
  const todayISO = toISODate(new Date());
  return Array.from({ length: 7 }, (_, i) => {
    const d = new Date(start);
    d.setDate(start.getDate() + i);
    const iso = toISODate(d);
    return { iso, dayKey: DAY_KEYS[d.getDay()]!, num: d.getDate(), isToday: iso === todayISO };
  });
}
