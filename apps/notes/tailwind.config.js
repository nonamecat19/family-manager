// Nocturne, as Commonplace draws it.
//
// The palette itself — the neutral and accent ramps, the three radii, and the repaint of the
// shared roles @fm/ui renders against — comes from `@fm/config/nocturne.preset.cjs`, shared
// with apps/finance, which is drawn from the same system. It is a preset of its own rather
// than an edit to `tailwind.preset.cjs`, because that one is shared with apps/recipes (drawn
// in Organic, a light system) and repainting it would repaint that app too.
//
// What stays here is what only THIS design uses: the two pane grounds, the second accent and
// the Inter faces. Values mirror `components/nocturne/tokens.ts`.
const accent2 = {
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
};

// The two pane grounds the design uses that are not on the ramp: the sidebar sits a shade
// under the app ground, the note list a shade over it. Named rather than left as literals in
// a screen, because "the rail colour" is a role this app repeats on every screen.
const rail = "#1b1d2c";
const list = "#191b29";

/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./app/**/*.{ts,tsx}", "./components/**/*.{ts,tsx}", "../../packages/ui/src/**/*.{ts,tsx}"],
  presets: [
    require("nativewind/preset"),
    require("@fm/config/tailwind.preset.cjs"),
    require("@fm/config/nocturne.preset.cjs"),
  ],
  theme: {
    extend: {
      colors: { accent2, rail, list },
      fontFamily: {
        // One face — Inter — in four weights, each its own TTF. The utilities are named per
        // weight rather than leaning on Tailwind's font-weight classes, because with
        // per-weight files `font-semibold` alone would not pick the right one. `strong`
        // rather than `bold` on purpose: `font-bold` is already a font-weight utility, and a
        // fontFamily key of the same name would collide with it.
        sans: ["Inter_400Regular"],
        med: ["Inter_500Medium"],
        semi: ["Inter_600SemiBold"],
        strong: ["Inter_700Bold"],
      },
    },
  },
};
