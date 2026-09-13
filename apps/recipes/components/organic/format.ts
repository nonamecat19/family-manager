import type { TranslationKey } from "../i18n/index.tsx";

export type TFunc = (key: TranslationKey, params?: Record<string, string | number>) => string;

export function formatDuration(seconds: number, t: TFunc): string {
  if (seconds <= 0) return "";
  const mins = Math.round(seconds / 60);
  if (mins < 60) return t("duration.minutes", { count: mins });
  const hrs = Math.floor(mins / 60);
  const rest = mins % 60;
  return rest === 0 ? t("duration.hours", { count: hrs }) : t("duration.hoursAndMinutes", { hours: hrs, minutes: rest });
}

export function formatTotalTime(seconds: number, t: TFunc): string {
  if (seconds <= 0) return t("duration.noTimeYet");
  const hrs = seconds / 3600;
  if (hrs < 1) return t("duration.minutesTotal", { count: Math.round(seconds / 60) });
  return t("duration.hoursTotal", { count: Math.round(hrs) });
}

export function metaLine(parts: (string | undefined)[]): string {
  return parts.filter((p) => p !== undefined && p !== "").join(" · ");
}

export function formatMacro(grams: number): string {
  return Number.isInteger(grams) ? String(grams) : grams.toFixed(1);
}
