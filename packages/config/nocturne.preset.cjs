// Nocturne — the dark design system apps/finance and apps/notes are both drawn from, as a
// Tailwind preset.
//
// It exists because the two apps' `tailwind.config.js` files carried this block verbatim:
// the same nine neutrals, the same nine accents, the same three radii, and the same repaint
// of the shared roles @fm/ui renders against. Two copies of a palette is how one app's
// surface quietly drifts a shade off the other's.
//
// This is a SEPARATE preset from `tailwind.preset.cjs`, not an edit to it: that one is shared
// with apps/recipes (`just impact pkg:@fm/config`), which is drawn in Organic — a light
// system — and repainting it there would repaint that app too. An app opts in by listing this
// after the base preset.
//
// The TypeScript mirror is `nocturneCore` in @fm/theme; `packages/theme/src/nocturne.test.ts`
// asserts the two never drift. This file must stay CommonJS, which is why the palette exists
// twice at all.

const neutral = {
  100: "#f3f5fe",
  200: "#e4e7f5",
  300: "#cfd3e5",
  400: "#b2b6ca",
  500: "#9397ab",
  600: "#75798c",
  700: "#595d6c",
  800: "#3f424d",
  900: "#292b31",
};

const accent = {
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
};

const bg = "#161826";
const surface = "#232532";
const text = "#e9e9ed";
const divider = "rgba(233,233,237,.16)";

// Nocturne carries no danger role of its own. The design adds exactly one value for it,
// generated in OKLCH at accent-400's own lightness and chroma (L 0.734, C 0.125, hue 27) so it
// sits on the same perceptual step as the rest of the ramp. In finance it means "over budget"
// and nothing else; it is also what both apps point `error` and `expense` at.
const overspend = "#e5928a";

/** @type {import('tailwindcss').Config} */
module.exports = {
  theme: {
    extend: {
      colors: {
        neutral,
        accent,
        overspend,
        // The shared roles @fm/ui renders against, repainted in Nocturne. Nocturne is
        // dark-only, so the `*-dark` counterparts resolve to the same values — a shared
        // component using `dark:` classes must not fall back to the preset's blue-greys.
        primary: { DEFAULT: accent.DEFAULT, fg: bg, muted: accent[900] },
        bg,
        surface,
        border: neutral[800],
        divider,
        fg: text,
        muted: neutral[500],
        "bg-dark": bg,
        "surface-dark": surface,
        "border-dark": neutral[800],
        "fg-dark": text,
        "muted-dark": neutral[500],
        error: overspend,
        expense: overspend,
      },
      borderRadius: {
        sm: "4px",
        md: "8px",
        lg: "14px",
      },
    },
  },
};
