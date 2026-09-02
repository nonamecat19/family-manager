/**
 * Nocturne — the dark design system `apps/finance` and `apps/notes` are both drawn from
 * (finance: design canvas `docs/design/finance/Family Money Manager.dc.html`, DS bundle
 * `nocturne-987014d8-7bfe-4008-8599-21473d96769b`; notes: the imported design's `styles.css`).
 *
 * This module is the part the two apps hold in COMMON, and it is common by measurement, not by
 * hope: the nine neutral steps, the nine accent steps, the three radii and the four ground
 * roles below were byte-identical in both apps' token files before they were lifted here.
 * Everything else each app carries — finance's header gradient, scrim, overspend hue and
 * spacing scale; notes' rail, list pane and second accent — stayed app-local, because it is
 * app-local: only one of the two designs draws it.
 *
 * Nocturne is DARK ONLY. There is no light pass of these screens; a component that needs a
 * "light" surface uses a lighter neutral, never a light theme. `lightScheme`/`darkScheme` in
 * `scheme.ts` are a different, older system and do not apply here.
 *
 * Each app mirrors what it uses into its own `tailwind.config.js`, which must stay CommonJS —
 * that is why the palette exists twice. Change a value here, change the mirrors.
 */
export const nocturneCore = {
  /** The app ground. Every screen sits on it; no screen paints its own background. */
  bg: "#161826",
  /** Cards, rows, list groups, the editor's block chrome. */
  surface: "#232532",
  text: "#e9e9ed",
  /** The signature rule. Notes draws it faded at both ends — see that app's `Divider`. */
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

  /** Blurple — the primary accent. Eight tonal steps plus the default. */
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

  /** Corner radii, the design's three steps. Mirrored in each app's `rounded-{sm,md,lg}`. */
  radius: { sm: 4, md: 8, lg: 14 },
} as const;

/** A colour pair for an avatar or icon circle: the ground and a foreground that survives it. */
export interface Tint {
  bg: string;
  fg: string;
}

/**
 * The uppercase initial an avatar shows for a person with no photo. Shared because "first
 * character, uppercased, `?` when there is nothing" is one rule, and two copies of it is how
 * one screen starts rendering an empty box where the other renders `?`.
 */
export function initialOf(text: string): string {
  const first = text.trim()[0];
  return first ? first.toUpperCase() : "?";
}
