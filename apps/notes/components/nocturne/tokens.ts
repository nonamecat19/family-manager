/**
 * Nocturne, as Commonplace draws it.
 *
 * The core — the neutral and accent ramps, the three radii, the ground/surface/text/divider
 * roles — lives in `@fm/theme` (`nocturneCore`), shared with apps/finance, which is drawn from
 * the same system. What stays here is what only THIS design uses: the two pane grounds the
 * three-column layout needs, and the muted second accent.
 *
 * The Tailwind mirror lives in `apps/notes/tailwind.config.js` — that file must stay CommonJS,
 * which is why the palette exists twice. Change one, change the other.
 *
 * Nocturne is DARK ONLY. There is no light pass of these screens; a component that needs a
 * "light" surface uses a lighter neutral, never a light theme.
 */
import { initialOf, nocturneCore, type Tint } from "@fm/theme";

export { initialOf, type Tint };

export const nocturne = {
  ...nocturneCore,

  /**
   * The sidebar's ground — a shade under `bg`, which is how the design separates navigation
   * from content without a border. A named token rather than a literal in a screen: it is a
   * role ("the rail"), and it appears on every screen that has one.
   */
  rail: "#1b1d2c",
  /** The note-list pane's ground, a shade over the rail and under `surface`. */
  list: "#191b29",

  /** The muted second accent: avatars, share chips, comment marks. */
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
} as const;

/** The type system: Inter, four weights, loaded per file in app/_layout.tsx. */
export const FONT_NOTE = "Inter; weights 400/500/600/700" as const;

/**
 * The tint a person is drawn in. Family members are a handful of names, not a fixed set, so a
 * name falls to a stable slot by hash rather than to one grey — two people in an avatar stack
 * must not be the same colour, and the same person must be the same colour on every screen.
 */
const MEMBER_TINTS: readonly Tint[] = [
  { bg: nocturne.accent[800], fg: nocturne.accent[200] },
  { bg: nocturne.accent2[700], fg: nocturne.accent2[100] },
  { bg: nocturne.accent[700], fg: nocturne.accent[100] },
  { bg: nocturne.accent2[800], fg: nocturne.accent2[200] },
  { bg: nocturne.neutral[800], fg: nocturne.neutral[200] },
];

/** Stable tint for a person; the same name always gets the same colour. */
export function tintFor(name: string | undefined, fallbackIndex = 0): Tint {
  const key = (name ?? "").trim().toLowerCase();
  const slot = key === "" ? fallbackIndex : hashString(key);
  return MEMBER_TINTS[slot % MEMBER_TINTS.length]!;
}

function hashString(s: string): number {
  let h = 0;
  for (let i = 0; i < s.length; i++) h = (h * 31 + s.charCodeAt(i)) >>> 0;
  return h;
}

