/** Durations are stored in seconds on the wire; the design only ever shows minutes/hours. */
export function formatDuration(seconds: number): string {
  if (seconds <= 0) return "";
  const mins = Math.round(seconds / 60);
  if (mins < 60) return `${mins} min`;
  const hrs = Math.floor(mins / 60);
  const rest = mins % 60;
  return rest === 0 ? `${hrs} h` : `${hrs} h ${rest} min`;
}

/** "3 h total" — the plan's summary line, which rounds rather than listing minutes. */
export function formatTotalTime(seconds: number): string {
  if (seconds <= 0) return "no time yet";
  const hrs = seconds / 3600;
  if (hrs < 1) return `${Math.round(seconds / 60)} min total`;
  return `${Math.round(hrs)} h total`;
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
