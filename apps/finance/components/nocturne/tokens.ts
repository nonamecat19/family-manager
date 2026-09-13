import { initialOf, nocturneCore, type Tint } from "@fm/theme";

export { initialOf, type Tint };

export const nocturne = {
  ...nocturneCore,

  headerGradient: ["#2b2741", "#242137"] as const,
  canvas: "#101120",
  scrim: "rgba(16,17,32,.55)",

  overspend: "#e5928a",

  space: { n1: 2.8, n2: 5.6, n3: 8.4, n4: 11.2, n5: 16.8, n6: 22.4 },
} as const;

export const FONT_NOTE = "system sans; weights 400/500/600" as const;

export const SERIES = [
  "#968ae0",
  "#4c5397",
  "#b5abfc",
  "#5d5294",
  "#7972a9",
  "#353b80",
  "#9397ab",
  "#595d6c",
] as const;

export function seriesColor(index: number): string {
  return SERIES[((index % SERIES.length) + SERIES.length) % SERIES.length]!;
}

export function memberColor(index: number): string {
  return seriesColor(index);
}

const DARK_ON = new Set(["#968ae0", "#b5abfc", "#9397ab"]);

export function tintFor(index: number): Tint {
  const bg = seriesColor(index);
  return { bg, fg: DARK_ON.has(bg) ? nocturne.bg : nocturne.accent[200] };
}

export interface BudgetState {
  ratio: number;
  percent: number;
  over: boolean;
}

export function budgetState(spentMinor: number, limitMinor: number): BudgetState | null {
  if (limitMinor <= 0) return null;
  const ratio = spentMinor / limitMinor;
  return { ratio, percent: Math.round(ratio * 100), over: ratio > 1 };
}
