import type { DateRange, PeriodInput } from "@fm/api";

import { dayHeading, monthNameGenitive, monthTitle, parseISO, type Translate } from "@/components/nocturne";

/**
 * The one line the design draws to the left of the period total — "Серпень 2026".
 *
 * Built from the kit's own month names rather than `Intl`: Hermes ships no ICU data for
 * Ukrainian, so `toLocaleDateString` would render an English month in the middle of a
 * Ukrainian screen.
 */
export function periodLabel(t: Translate, period: PeriodInput, window: DateRange): string {
  switch (period.granularity) {
    case "day":
      return dayHeading(t, window.from);
    case "month": {
      const { year, month } = parseISO(window.from);
      return monthTitle(t, year, month);
    }
    case "year":
      return `${parseISO(window.from).year}`;
    default:
      return rangeLabel(t, window);
  }
}

/** "24 – 30 серпня", and "28 серпня – 3 вересня" across a month boundary. */
function rangeLabel(t: Translate, window: DateRange): string {
  const from = parseISO(window.from);
  const to = parseISO(window.to);
  const toLabel = `${to.day} ${monthNameGenitive(t, to.month)}`;
  if (from.month === to.month && from.year === to.year) return `${from.day} – ${toLabel}`;
  return `${from.day} ${monthNameGenitive(t, from.month)} – ${toLabel}`;
}
