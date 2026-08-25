/** ISO calendar day (YYYY-MM-DD) — the meal plan's own date format, not a timestamp. */
export function toISODate(d: Date): string {
  const year = d.getFullYear();
  const month = `${d.getMonth() + 1}`.padStart(2, "0");
  const day = `${d.getDate()}`.padStart(2, "0");
  return `${year}-${month}-${day}`;
}

/**
 * The Sunday-to-Saturday week `date` falls in. Built from local date parts rather than
 * `toISOString()`, which would shift the day for anyone east or west of UTC and quietly plan
 * meals for the wrong Sunday.
 */
export function weekRange(date: Date): { from: string; to: string } {
  const start = new Date(date);
  start.setDate(date.getDate() - date.getDay());
  const end = new Date(start);
  end.setDate(start.getDate() + 6);
  return { from: toISODate(start), to: toISODate(end) };
}

export interface WeekDay {
  iso: string;
  /** "Mon" — the uppercase label under the date number. */
  day: string;
  /** The day of the month. */
  num: number;
  isToday: boolean;
}

const DAY_NAMES = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];

export function buildWeek(date: Date): WeekDay[] {
  const start = new Date(date);
  start.setDate(date.getDate() - date.getDay());
  const todayISO = toISODate(new Date());
  return Array.from({ length: 7 }, (_, i) => {
    const d = new Date(start);
    d.setDate(start.getDate() + i);
    const iso = toISODate(d);
    return { iso, day: DAY_NAMES[d.getDay()]!, num: d.getDate(), isToday: iso === todayISO };
  });
}
