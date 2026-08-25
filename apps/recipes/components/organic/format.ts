import type { TranslationKey } from "../i18n/index.tsx";

/** The shape of `useI18n().t` — passed in rather than called via the hook, since these are
 * plain functions, not components. */
export type TFunc = (key: TranslationKey, params?: Record<string, string | number>) => string;

/** Durations are stored in seconds on the wire; the design only ever shows minutes/hours. */
export function formatDuration(seconds: number, t: TFunc): string {
  if (seconds <= 0) return "";
  const mins = Math.round(seconds / 60);
  if (mins < 60) return t("duration.minutes", { count: mins });
  const hrs = Math.floor(mins / 60);
  const rest = mins % 60;
  return rest === 0 ? t("duration.hours", { count: hrs }) : t("duration.hoursAndMinutes", { hours: hrs, minutes: rest });
}

/** "3 h total" — the plan's summary line, which rounds rather than listing minutes. */
export function formatTotalTime(seconds: number, t: TFunc): string {
  if (seconds <= 0) return t("duration.noTimeYet");
  const hrs = seconds / 3600;
  if (hrs < 1) return t("duration.minutesTotal", { count: Math.round(seconds / 60) });
  return t("duration.hoursTotal", { count: Math.round(hrs) });
}

/** "Lunch · Soup · 45 min", skipping whatever the recipe doesn't have. */
export function metaLine(parts: (string | undefined)[]): string {
  return parts.filter((p) => p !== undefined && p !== "").join(" · ");
}

/** "18.5" but "48", not "48.0" — book macros are printed both ways and the trailing zero
 * makes the strip look noisier than the numbers deserve. */
export function formatMacro(grams: number): string {
  return Number.isInteger(grams) ? String(grams) : grams.toFixed(1);
}
