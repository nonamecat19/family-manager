/**
 * Nocturne — the design system this app is drawn from (design canvas
 * `docs/design/finance/Family Money Manager.dc.html`, DS bundle
 * `nocturne-987014d8-7bfe-4008-8599-21473d96769b`).
 *
 * These values are finance-only on purpose: @fm/theme and @fm/config are shared with
 * apps/recipes, so retuning them there would repaint that app too. The Tailwind mirror lives
 * in `apps/finance/tailwind.config.js` — that file must stay CommonJS, which is why the
 * palette exists twice. Change one, change the other.
 *
 * Nocturne is DARK ONLY. There is no light pass of these screens; a component that needs a
 * "light" surface uses a lighter neutral, never a light theme.
 */

export const nocturne = {
  /** The app ground. Every screen sits on it; no screen paints its own background. */
  bg: "#161826",
  /** Cards, rows, list groups. */
  surface: "#232532",
  /** The header's gradient, top to bottom. */
  headerGradient: ["#2b2741", "#242137"] as const,
  text: "#e9e9ed",
  divider: "rgba(233,233,237,.16)",
  /** Behind the phone frame on the canvas; also the scrim base. */
  canvas: "#101120",
  scrim: "rgba(16,17,32,.55)",

  neutral: {
    100: "#f3f5fe",
    200: "#e4e7f5",
    300: "#cfd3e5",
    400: "#b2b6ca",
    500: "#9397ab",
    600: "#75798c",
    700: "#595d6c",
    800: "#3f424d",
    900: "#292b31",
  },

  /** Blurple — the only hue in the system. Eight tonal steps carry the whole chart language. */
  accent: {
    DEFAULT: "#9184d9",
    100: "#f5f4ff",
    200: "#e7e5fe",
    300: "#d2cefd",
    400: "#b5abfc",
    500: "#968ae0",
    600: "#796cbf",
    700: "#5d5294",
    800: "#423a6a",
    900: "#2b2741",
  },

  /**
   * The one value added to Nocturne: a budget breach. Generated in OKLCH at accent-400's own
   * lightness and chroma (L 0.734, C 0.125, hue 27) so it sits on the same perceptual step as
   * the rest of the ramp. It means "over budget" and nothing else — it is not a generic error
   * colour, and an expense that is merely large is not drawn in it.
   */
  overspend: "#e5928a",

  radius: { sm: 4, md: 8, lg: 14 },

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
export interface Tint {
  bg: string;
  fg: string;
}

const DARK_ON = new Set(["#968ae0", "#b5abfc", "#9397ab"]);

/** Icon-circle tint for the nth group or category. Light steps get the dark ground back. */
export function tintFor(index: number): Tint {
  const bg = seriesColor(index);
  return { bg, fg: DARK_ON.has(bg) ? nocturne.bg : nocturne.accent[200] };
}

/** The uppercase initial an avatar shows for a member with no photo. */
export function initialOf(text: string): string {
  const first = text.trim()[0];
  return first ? first.toUpperCase() : "?";
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
