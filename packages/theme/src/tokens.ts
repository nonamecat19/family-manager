/**
 * Design tokens. These mirror `@fm/config/tailwind.preset.cjs`; `tokens.test.ts` asserts the
 * two never drift. Tailwind config must stay CommonJS, which is why the values exist twice
 * instead of being imported.
 */

export const colors = {
  primary: "#2F855A",
  primaryFg: "#FFFFFF",
  primaryMuted: "#C6F6D5",

  /** Income is green, expense is red, transfer is blue — fixed, never themed away. */
  income: "#2F855A",
  expense: "#C53030",
  transfer: "#2B6CB0",

  bg: "#F7FAFC",
  surface: "#FFFFFF",
  border: "#E2E8F0",
  fg: "#1A202C",
  muted: "#718096",

  bgDark: "#12161C",
  surfaceDark: "#1A202C",
  borderDark: "#2D3748",
  fgDark: "#F7FAFC",
  mutedDark: "#A0AEC0",
} as const;

export const spacing = {
  xs: 4,
  sm: 8,
  md: 12,
  lg: 16,
  xl: 24,
  "2xl": 32,
} as const;

export const radius = {
  sm: 6,
  md: 10,
  lg: 16,
  full: 9999,
} as const;

export const fontSize = {
  caption: 12,
  body: 15,
  title: 20,
  display: 32,
  amount: 28,
} as const;

/**
 * The palette assigned to categories in charts, in order. A category with no colour of its
 * own gets `categoryPalette[index % categoryPalette.length]`, so the same category keeps the
 * same slice colour across renders.
 */
export const categoryPalette = [
  "#2F855A",
  "#C53030",
  "#2B6CB0",
  "#B7791F",
  "#6B46C1",
  "#00A3C4",
  "#DD6B20",
  "#319795",
  "#D53F8C",
  "#4A5568",
] as const;

export type ColorToken = keyof typeof colors;
export type SpacingToken = keyof typeof spacing;
