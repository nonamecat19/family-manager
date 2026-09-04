/**
 * Nocturne, as apps/finance draws it.
 *
 * The core — the neutral and accent ramps, the three radii, the ground/surface/text/divider
 * roles — lives in `@fm/theme` (`nocturneCore`), shared with apps/notes, which is drawn from
 * the same system. What stays here is what only THIS design uses: the header gradient, the
 * canvas and scrim behind the phone frame, the overspend hue, and the spacing scale.
 *
 * The Tailwind mirror lives in `apps/finance/tailwind.config.js` — that file must stay
 * CommonJS, which is why the palette exists twice. Change one, change the other.
 *
 * Nocturne is DARK ONLY. There is no light pass of these screens; a component that needs a
 * "light" surface uses a lighter neutral, never a light theme.
 */
import { initialOf, nocturneCore, type Tint } from "@fm/theme";

export { initialOf, type Tint };

export const nocturne = {
  ...nocturneCore,

  /** The header's gradient, top to bottom. */
  headerGradient: ["#2b2741", "#242137"] as const,
  /** Behind the phone frame on the canvas; also the scrim base. */
  canvas: "#101120",
  scrim: "rgba(16,17,32,.55)",

  /**
   * The one value added to Nocturne: a budget breach. Generated in OKLCH at accent-400's own
   * lightness and chroma (L 0.734, C 0.125, hue 27) so it sits on the same perceptual step as
   * the rest of the ramp. It means "over budget" and nothing else — it is not a generic error
   * colour, and an expense that is merely large is not drawn in it.
   */
  overspend: "#e5928a",

  /** Nocturne's spacing scale, mirrored by the `n1`…`n6` Tailwind steps. */
  space: { n1: 2.8, n2: 5.6, n3: 8.4, n4: 11.2, n5: 16.8, n6: 22.4 },
} as const;

/**
 * Inter is the design's face, but the app ships no font file: adding a webfont package is a
 * dependency this repo's stack table does not carry, and Nocturne's own stack ends in
 * `system-ui, sans-serif`. Screens therefore use weight utilities (`font-normal`,
 * `font-medium`, `font-semibold`) and the platform sans — Roboto on Android reads at the same
 * sizes. Nothing sets `fontFamily`.
 */
export const FONT_NOTE = "system sans; weights 400/500/600" as const;

/**
 * The donut/stack series ramp, in the exact order the design's conic-gradient uses. Segments
 * are assigned by index, not by category name, so the same chart drawn twice looks the same
 * and a family that renames a group does not reshuffle its colours.
 */
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

/** Colour for the nth series slot; wraps, so an eleven-group family still gets a chart. */
export function seriesColor(index: number): string {
  return SERIES[((index % SERIES.length) + SERIES.length) % SERIES.length]!;
}

/**
 * A member's colour. Members are ordered by the caller (household order, stable) and the
 * first two land on the two values the design uses for Сергій and Олена.
 */
export function memberColor(index: number): string {
  return seriesColor(index);
}

/** The tint of a category/group avatar: the ramp step plus a foreground that survives it. */
const DARK_ON = new Set(["#968ae0", "#b5abfc", "#9397ab"]);

/** Icon-circle tint for the nth group or category. Light steps get the dark ground back. */
export function tintFor(index: number): Tint {
  const bg = seriesColor(index);
  return { bg, fg: DARK_ON.has(bg) ? nocturne.bg : nocturne.accent[200] };
}

/**
 * Where a budget stands. `ratio` is spend ÷ limit, unclamped, so a caller can print 256%;
 * `over` is what turns the bar and the figure to `overspend`. A limit of 0 means "no budget",
 * which is not the same as "spent nothing" — it returns null so a bar is not drawn at all.
 */
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
