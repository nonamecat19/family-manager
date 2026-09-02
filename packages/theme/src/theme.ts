/**
 * The theme contract: one shape that every design system in this repo satisfies, so a shared
 * component can be drawn once and take its paint from whichever app is rendering it.
 *
 * The two systems here look nothing alike — Nocturne is a tight, dark, blurple system with
 * 4/8/14 radii; Organic is a warm, light, terracotta one whose corners run 20/26/36 — and the
 * point of this file is that the DIFFERENCE is data. A component reads `theme.surface` and
 * `theme.radius.md` and comes out looking native to either, which is what makes "one design
 * system, different themes" true rather than aspirational.
 *
 * What does NOT belong here: anything only one system has. A role earns a slot on this
 * interface when both systems can answer it honestly; a value only Nocturne draws (its header
 * gradient) or only Organic draws (its category tints) stays in the app.
 */

/** A nine-step tonal ramp, light to dark. */
export interface Ramp {
  100: string;
  200: string;
  300: string;
  400: string;
  500: string;
  600: string;
  700: string;
  800: string;
  900: string;
}

/** A ramp with the step the system reaches for when no tone is named. */
export interface AccentRamp extends Ramp {
  DEFAULT: string;
}

export type SchemeName = "dark" | "light";

export interface Theme {
  /** Which system this is. Present so a component can make a rare, deliberate exception. */
  name: string;
  /**
   * Whether the ground is dark or light. Components use this for the things a palette cannot
   * express on its own — the status bar style, a shadow that only reads on light.
   */
  scheme: SchemeName;

  /** The app ground. Every screen sits on it; no screen paints its own background. */
  bg: string;
  /** Cards, rows, list groups. */
  surface: string;
  /** Body text on `bg` and `surface`. */
  text: string;
  /** Secondary text: labels, timestamps, the things that must recede. */
  muted: string;
  /** Hairline rules and card edges. */
  divider: string;

  neutral: Ramp;
  accent: AccentRamp;
  /** The secondary accent. A system with only one hue points this at `accent`. */
  accent2: AccentRamp;

  /** Destructive actions and error states. */
  danger: string;
  /** Text and icons drawn ON `accent.DEFAULT` — never assume white. */
  accentFg: string;
  /** Text and icons drawn ON `danger`. */
  dangerFg: string;

  /** The system's three corner steps. Their absolute values differ wildly between systems. */
  radius: { sm: number; md: number; lg: number };

  /**
   * The faces a component draws in, by role.
   *
   * Optional, and undefined means the platform face — which is a real choice, not a gap:
   * apps/finance ships no font file and Nocturne's own stack ends in `system-ui, sans-serif`.
   *
   * Fonts sit on the THEME rather than the base system because two apps can share a palette
   * and not a typeface: finance and notes are both Nocturne, but notes loads Inter and finance
   * does not. An app supplies these by spreading a base theme and adding its own, which is why
   * `ThemeProvider` takes a theme object and not just a name.
   *
   * Values are React Native font family names — for per-weight TTFs (`Inter_600SemiBold`) that
   * is the loaded name, not a CSS stack.
   */
  /**
   * How a text input is drawn. This is a THEME decision, not a component one: Nocturne draws
   * finance's inputs as a bare rule under the text, notes' as an outlined dark surface, and
   * Organic draws recipes' as a filled pill. A shared `Field` that picked one would silently
   * restyle two of the three apps.
   *
   * Defaults to "outline" when a theme does not say.
   */
  fieldStyle?: "underline" | "outline" | "filled";

  /**
   * How a field or section label is set. Nocturne sets them as small uppercase with tracking;
   * Organic sets them in sentence case. Same reason as `fieldStyle`: picking one in the
   * component would restyle whichever app did not get picked.
   *
   * Defaults to "uppercase".
   */
  labelCase?: "uppercase" | "sentence";

  fonts?: {
    body?: string;
    medium?: string;
    semibold?: string;
    /** Headings and display type, where a system uses a second face. */
    display?: string;
  };
}
