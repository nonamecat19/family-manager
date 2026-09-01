import type { Timestamp } from "@bufbuild/protobuf/wkt";

/**
 * Time formatting, in one place for the same reason the copy is: the list row, the editor
 * header and the activity rail all print "how long ago", and three hand-rolled versions of
 * that is how a screen ends up saying "4m ago" next to "4 minutes ago".
 */

/** The wire's Timestamp as a Date. Returns null for an absent timestamp. */
export function toDate(ts: Timestamp | undefined): Date | null {
  if (!ts) return null;
  return new Date(Number(ts.seconds) * 1000 + Math.floor(ts.nanos / 1e6));
}

const MINUTE = 60_000;
const HOUR = 60 * MINUTE;
const DAY = 24 * HOUR;

/** "just now" · "4m ago" · "3h ago" · "yesterday" · "12 Aug". The list row's line. */
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

/** The dense list's right-hand stamp: "4m" · "9:12" · "Mon" · "26 Aug". */
export function stamp(ts: Timestamp | undefined, now = Date.now()): string {
  const date = toDate(ts);
  if (!date) return "";
  const delta = now - date.getTime();
  if (delta < HOUR) return `${Math.max(1, Math.floor(delta / MINUTE))}m`;
  if (delta < DAY) return clock(date);
  if (delta < 7 * DAY) return weekday(date);
  return shortDate(date);
}

/** "14 Aug" — the editor header's created line. */
export function shortDate(date: Date): string {
  return `${date.getDate()} ${MONTHS[date.getMonth()] ?? ""}`;
}

export function longDate(ts: Timestamp | undefined): string {
  const date = toDate(ts);
  if (!date) return "";
  return `${date.getDate()} ${MONTHS[date.getMonth()] ?? ""} ${date.getFullYear()}`;
}

/** Which of the note list's three groups a note falls into. */
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
