import { initialOf, organic, type Tint } from "@fm/ui";

export { initialOf, organic, type Tint };

/**
 * Categorical chart palette. Derived from the organic ramps wherever the ramps reach far enough
 * (`accent`, `accent2`, `neutral`, `danger`); the ochre and slate stops are the two hues the
 * ramps do not carry and are defined here, in the token module — the one place literals live.
 * Ordered so neighbouring series indices alternate hue and lightness; every stop clears 3:1
 * against the organic `bg` and `surface` ramps (WCAG non-text minimum).
 */
export const SERIES = [
  organic.accent.DEFAULT, // 0 terracotta
  organic.accent2.DEFAULT, // 1 olive
  organic.accent[700], // 2 burnt umber
  organic.accent2[700], // 3 dark olive
  organic.neutral[700], // 4 warm taupe
  organic.danger, // 5 brick
  "#9a6b1f", // 6 ochre
  "#3f5d6e", // 7 slate
] as const;

export function seriesColor(index: number): string {
  return SERIES[((index % SERIES.length) + SERIES.length) % SERIES.length]!;
}

export function memberColor(index: number): string {
  return seriesColor(index);
}

/**
 * Soft tint pairs for icon bubbles. Finance picks tints by a numeric `colorStep`, so the shared
 * name-hashed `tintFor` from `@fm/ui` does not apply here — this is the index-keyed counterpart.
 */
const TINTS = [
  { bg: organic.accent2[200], fg: organic.accent2[800] },
  { bg: organic.accent[200], fg: organic.accent[800] },
  { bg: organic.accent2[300], fg: organic.accent2[900] },
  { bg: organic.accent[300], fg: organic.accent[900] },
  { bg: organic.neutral[200], fg: organic.neutral[900] },
  { bg: organic.accent2[400], fg: organic.accent2[900] },
  { bg: organic.accent[400], fg: organic.accent[900] },
  { bg: organic.neutral[300], fg: organic.neutral[900] },
] as const;

export function tintFor(index: number): Tint {
  return TINTS[((index % TINTS.length) + TINTS.length) % TINTS.length]!;
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
