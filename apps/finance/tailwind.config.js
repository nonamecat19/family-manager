// Nocturne, as apps/finance draws it.
//
// The palette itself — the neutral and accent ramps, the three radii, and the repaint of the
// shared roles @fm/ui renders against — comes from `@fm/config/nocturne.preset.cjs`, shared
// with apps/notes, which is drawn from the same system. It is a preset of its own rather than
// an edit to `tailwind.preset.cjs`, because that one is shared with apps/recipes (drawn in
// Organic, a light system) and repainting it would repaint that app too.
//
// What stays here is what only THIS design uses: the canvas behind the phone frame and the
// n1…n6 spacing scale. Values mirror `components/nocturne/tokens.ts`.
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
      colors: {
        // The ground the design canvas itself sits on, behind the phone frame.
        canvas: "#101120",
      },
      // Nocturne's spacing scale. Named n1…n6 so they never collide with Tailwind's numeric
      // spacing steps, which screens still use for one-off gaps.
      spacing: {
        n1: "2.8px",
        n2: "5.6px",
        n3: "8.4px",
        n4: "11.2px",
        n5: "16.8px",
        n6: "22.4px",
      },
    },
  },
};
