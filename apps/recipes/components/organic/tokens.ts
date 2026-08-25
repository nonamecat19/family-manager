/**
 * Organic — the design system this app is drawn from (Claude Design project
 * `Family Recipes App.dc.html`, DS bundle `organic-0a9cdade`).
 *
 * These values are recipes-only on purpose: @fm/theme and @fm/config are shared with
 * apps/finance, so retuning them there would repaint the finance app too. The Tailwind
 * mirror lives in `apps/recipes/tailwind.config.js` — that file must stay CommonJS, which
 * is why the palette exists twice. Change one, change the other.
 */

export const organic = {
  bg: "#f5ead8",
  surface: "#ebddc5",
  text: "#201e1d",
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

  /** Terracotta — the primary accent: buttons, active pills, ratings. */
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

  /** Olive — the secondary accent: the plan, notes, step numbers. */
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

  /** Destructive actions. Organic has no error role, so this is the palette's warm red. */
  danger: "#a5341f",
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
