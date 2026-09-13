import type { DateRange, PeriodInput } from "@fm/api";

import { dayHeading, monthNameGenitive, monthTitle, parseISO, type Translate } from "@/components/nocturne";

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

function rangeLabel(t: Translate, window: DateRange): string {
  const from = parseISO(window.from);
  const to = parseISO(window.to);
  const toLabel = `${to.day} ${monthNameGenitive(t, to.month)}`;
  if (from.month === to.month && from.year === to.year) return `${from.day} – ${toLabel}`;
  return `${from.day} ${monthNameGenitive(t, from.month)} – ${toLabel}`;
}
