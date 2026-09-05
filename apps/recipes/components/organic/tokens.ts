/**
 * Organic, as apps/recipes draws it (Claude Design project `Family Recipes App.dc.html`, DS
 * bundle `organic-0a9cdade`).
 *
 * The palette itself is `organicTheme` in `@fm/theme` — one of the two themes the shared
 * component library renders against. It lives there rather than here so a component in
 * `@fm/ui` can be drawn in Organic without importing an app, which is the one boundary rule
 * that never bends.
 *
 * What stays here is what only this app draws: the category tints and the name-to-tint rule.
 *
 * The Tailwind mirror lives in `apps/recipes/tailwind.config.js` — that file must stay
 * CommonJS, which is why the palette exists twice. `themes.test.ts` in @fm/theme asserts the
 * two never drift.
 */
import { organicTheme } from "@fm/theme";

/**
 * The app's token object. Shaped exactly as it was when the values lived here, so no screen
 * changed: `organic.accent[700]`, `organic.divider` and the rest all still resolve.
 */
export const organic = {
  bg: organicTheme.bg,
  // Note: this role previously carried #ebddc5 here while the Tailwind mirror resolved it to
  // neutral-100. No screen read either, so they are one value now — the Tailwind one.
  surface: organicTheme.surface,
  text: organicTheme.text,
  divider: organicTheme.divider,
  muted: organicTheme.muted,
  neutral: organicTheme.neutral,
  accent: organicTheme.accent,
  accent2: organicTheme.accent2,
  danger: organicTheme.danger,
} as const;

/**
 * The tint a category is drawn in. Categories are family-authored strings, so a name that
 * isn't one of the four the design names falls through to a stable slot by index rather
 * than to a single grey — a cookbook with "Zupy" and "Ciasta" still gets a coloured grid.
 */
const CATEGORY_TINTS = [
  { bg: organic.accent2[200], fg: organic.accent2[800] }, // Breakfast
  { bg: organic.accent[200], fg: organic.accent[800] }, // Lunch
  { bg: organic.accent2[300], fg: organic.accent2[900] }, // Dinner
  { bg: organic.accent[300], fg: organic.accent[900] }, // Dessert
] as const;

export interface Tint {
  bg: string;
  fg: string;
}

const NAMED_TINTS: Record<string, number> = {
  breakfast: 0,
  lunch: 1,
  dinner: 2,
  dessert: 3,
};

/** Stable tint for a category name; the same name always gets the same colour. */
export function tintFor(name: string | undefined, fallbackIndex = 0): Tint {
  const key = (name ?? "").trim().toLowerCase();
  const named = NAMED_TINTS[key];
  if (named !== undefined) return CATEGORY_TINTS[named]!;
  const hash = key === "" ? fallbackIndex : hashString(key);
  return CATEGORY_TINTS[hash % CATEGORY_TINTS.length]!;
}

function hashString(s: string): number {
  let h = 0;
  for (let i = 0; i < s.length; i++) h = (h * 31 + s.charCodeAt(i)) >>> 0;
  return h;
}

/** The uppercase initial a card shows when a recipe has no photo. */
export function initialOf(text: string): string {
  const first = text.trim()[0];
  return first ? first.toUpperCase() : "?";
}
