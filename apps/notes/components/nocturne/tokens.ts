/**
 * Nocturne — the design system Commonplace is drawn from (the imported design's
 * `styles.css`; the same system apps/finance carries, extended here with the second accent
 * and the two pane grounds this app's three-column layout needs).
 *
 * These values are notes-only on purpose: @fm/theme and @fm/config are shared with the other
 * apps, so retuning them there would repaint those too. The Tailwind mirror lives in
 * `apps/notes/tailwind.config.js` — that file must stay CommonJS, which is why the palette
 * exists twice. Change one, change the other.
 *
 * Nocturne is DARK ONLY. There is no light pass of these screens; a component that needs a
 * "light" surface uses a lighter neutral, never a light theme.
 */

export const nocturne = {
  /** The app ground. Every screen sits on it; no screen paints its own background. */
  bg: "#161826",
  /** Cards, rows, list groups, the editor's block chrome. */
  surface: "#232532",
  /**
   * The sidebar's ground — a shade under `bg`, which is how the design separates navigation
   * from content without a border. A named token rather than a literal in a screen: it is a
   * role ("the rail"), and it appears on every screen that has one.
   */
  rail: "#1b1d2c",
  /** The note-list pane's ground, a shade over the rail and under `surface`. */
  list: "#191b29",
  text: "#e9e9ed",
  /** The signature rule. It is drawn faded at both ends — see `Divider` in ui.tsx. */
  divider: "rgba(233,233,237,.16)",

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

  /** Blurple — the primary accent: the primary button's outline, stars, active rail rows. */
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

  /** Corner radii, the design's three steps. Mirrored in tailwind's `rounded-{sm,md,lg}`. */
  radius: {
    sm: 4,
    md: 8,
    lg: 14,
  },
} as const;

/** The type system: Inter, four weights, loaded per file in app/_layout.tsx. */
export const FONT_NOTE = "Inter; weights 400/500/600/700" as const;

export interface Tint {
  bg: string;
  fg: string;
}

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

/** The uppercase initial an avatar shows. */
export function initialOf(text: string): string {
  const first = text.trim()[0];
  return first ? first.toUpperCase() : "?";
}
