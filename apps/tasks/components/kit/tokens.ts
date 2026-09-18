import { initialOf, organic, type Tint } from "@fm/ui";

export { initialOf, organic, type Tint };

/**
 * Categorical chart palette for tasks. Uses the organic ramps.
 */
export const SERIES = [
  organic.accent.DEFAULT, // 0 terracotta
  organic.accent2.DEFAULT, // 1 olive
  organic.accent[700], // 2 burnt umber
  organic.accent2[700], // 3 dark olive
  organic.neutral[700], // 4 warm taupe
  organic.danger, // 5 brick
] as const;

export function seriesColor(index: number): string {
  return SERIES[((index % SERIES.length) + SERIES.length) % SERIES.length]!;
}

export function memberColor(index: number): string {
  return seriesColor(index);
}

/**
 * Soft tint pairs for icon bubbles.
 */
const TINTS = [
  { bg: organic.accent2[200], fg: organic.accent2[800] },
  { bg: organic.accent[200], fg: organic.accent[800] },
  { bg: organic.accent2[300], fg: organic.accent2[900] },
  { bg: organic.accent[300], fg: organic.accent[900] },
  { bg: organic.neutral[200], fg: organic.neutral[900] },
] as const;

export function tintFor(index: number): Tint {
  return TINTS[((index % TINTS.length) + TINTS.length) % TINTS.length]!;
}