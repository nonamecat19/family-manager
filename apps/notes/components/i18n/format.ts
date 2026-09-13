import type { Timestamp } from "@bufbuild/protobuf/wkt";


export function toDate(ts: Timestamp | undefined): Date | null {
  if (!ts) return null;
  return new Date(Number(ts.seconds) * 1000 + Math.floor(ts.nanos / 1e6));
}

const MINUTE = 60_000;
const HOUR = 60 * MINUTE;
const DAY = 24 * HOUR;

export function relative(ts: Timestamp | undefined, now = Date.now()): string {
  const date = toDate(ts);
  if (!date) return "";
  const delta = now - date.getTime();
  if (delta < MINUTE) return "just now";
  if (delta < HOUR) return `${Math.floor(delta / MINUTE)}m ago`;
  if (delta < DAY) return `${Math.floor(delta / HOUR)}h ago`;
  if (delta < 2 * DAY) return "yesterday";
  if (delta < 7 * DAY) return weekday(date);
  return shortDate(date);
}

export function stamp(ts: Timestamp | undefined, now = Date.now()): string {
  const date = toDate(ts);
  if (!date) return "";
  const delta = now - date.getTime();
  if (delta < HOUR) return `${Math.max(1, Math.floor(delta / MINUTE))}m`;
  if (delta < DAY) return clock(date);
  if (delta < 7 * DAY) return weekday(date);
  return shortDate(date);
}

export function shortDate(date: Date): string {
  return `${date.getDate()} ${MONTHS[date.getMonth()] ?? ""}`;
}

export function longDate(ts: Timestamp | undefined): string {
  const date = toDate(ts);
  if (!date) return "";
  return `${date.getDate()} ${MONTHS[date.getMonth()] ?? ""} ${date.getFullYear()}`;
}

export function bucket(ts: Timestamp | undefined, now = Date.now()): "today" | "week" | "older" {
  const date = toDate(ts);
  if (!date) return "older";
  const delta = now - date.getTime();
  if (delta < DAY) return "today";
  if (delta < 7 * DAY) return "week";
  return "older";
}

const MONTHS = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];
const WEEKDAYS = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];

function weekday(date: Date): string {
  return WEEKDAYS[date.getDay()] ?? "";
}

function clock(date: Date): string {
  return `${date.getHours()}:${String(date.getMinutes()).padStart(2, "0")}`;
}
