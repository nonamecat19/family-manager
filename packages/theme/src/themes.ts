/**
 * The two themes, each built from the palette its app already ships.
 *
 * Nocturne's shared core lives in `nocturne.ts` (it is mirrored by
 * `@fm/config/nocturne.preset.cjs` and guarded by `nocturne.test.ts`); the values added here
 * are the roles the theme contract asks for that the core does not carry.
 *
 * Organic's palette lives HERE rather than in apps/recipes, and the app's
 * `components/organic/tokens.ts` now derives from it. A package may not import an app, so the
 * only way to have one copy instead of two is for the package to own it — the same move the
 * Nocturne core made. The app keeps its own `tailwind.config.js` mirror, which is CommonJS and
 * therefore still a second copy; `themes.test.ts` guards that pair.
 */
import { nocturneCore } from "./nocturne.ts";
import type { Theme } from "./theme.ts";

/** Nocturne — apps/finance and apps/notes. Dark only; there is no light pass of it. */
export const nocturneTheme: Theme = {
  name: "nocturne",
  scheme: "dark",

  bg: nocturneCore.bg,
  surface: nocturneCore.surface,
  text: nocturneCore.text,
  muted: nocturneCore.neutral[500],
  divider: nocturneCore.divider,

  neutral: nocturneCore.neutral,
  accent: nocturneCore.accent,
  /** The muted second accent apps/notes draws avatars and share chips in. */
  accent2: {
    DEFAULT: "#a7a1db",
    100: "#f5f4ff",
    200: "#e7e5fe",
    300: "#d2cefd",
    400: "#b5afe8",
    500: "#9690c9",
    600: "#7972a9",
    700: "#5c5783",
    800: "#423e5d",
    900: "#2b293a",
  },

  // The one value added to Nocturne, generated in OKLCH on the accent ramp's own step. It is
  // what finance draws a budget breach in and what both apps point `error` at.
  danger: "#e5928a",
  // Nocturne's accent is light enough that the app ground is what reads on top of it — this
  // is the `primary.fg` its Tailwind preset already resolves to.
  accentFg: nocturneCore.bg,
  dangerFg: nocturneCore.bg,

  radius: nocturneCore.radius,
  // apps/notes draws an outlined dark surface; apps/finance overrides this to "underline",
  // which is what its own screens are designed around.
  fieldStyle: "outline",
};

/** Organic — apps/recipes. A warm, light system with much softer corners. */
export const organicTheme: Theme = {
  name: "organic",
  scheme: "light",

  bg: "#f5ead8",
  // neutral-100, matching the `surface` role the app's Tailwind config resolves — the value
  // shared components render against. The app's old token module carried a second, different
  // colour under this name (#ebddc5) that no screen ever used; the two are one now.
  surface: "#f9f4ed",
  text: "#201e1d",
  muted: "#82796a",
  divider: "rgba(32,30,29,0.16)",

  neutral: {
    100: "#f9f4ed",
    200: "#eee7db",
    300: "#dcd3c4",
    400: "#c0b6a5",
    500: "#a19786",
    600: "#82796a",
    700: "#645c50",
    800: "#474238",
    900: "#2e2b25",
  },

  /** Terracotta — buttons, active pills, ratings. */
  accent: {
    DEFAULT: "#c67139",
    100: "#fff2eb",
    200: "#ffe1d0",
    300: "#ffc6a5",
    400: "#f6a06b",
    500: "#d67f48",
    600: "#b2622d",
    700: "#8c491a",
    800: "#643312",
    900: "#402310",
  },

  /** Olive — the plan, notes, step numbers. */
  accent2: {
    DEFAULT: "#7a8a5e",
    100: "#f0fae1",
    200: "#e1eecc",
    300: "#ccdbb2",
    400: "#aebf92",
    500: "#8fa073",
    600: "#728157",
    700: "#56633f",
    800: "#3d472b",
    900: "#272e1b",
  },

  danger: "#a5341f",
  // Organic's accent is saturated enough to carry white — which is what the app's own
  // PrimaryButton draws on it.
  accentFg: "#ffffff",
  dangerFg: "#ffffff",

  // Organic's three steps are its `xl`/`2xl`/`3xl`: this is a pill-shaped system, and a
  // component asking for `radius.md` must get Organic's idea of a corner, not Nocturne's.
  radius: { sm: 20, md: 26, lg: 36 },
  fieldStyle: "filled",
  labelCase: "sentence",
};

export const themes = { nocturne: nocturneTheme, organic: organicTheme } as const;

export type ThemeName = keyof typeof themes;
