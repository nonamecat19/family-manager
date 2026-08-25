// Organic — the recipes app's own palette. It deliberately overrides the shared
// @fm/config preset's brand/surface roles instead of editing the preset, because the preset
// is also apps/finance's palette (see `just impact pkg:@fm/config`). Values mirror
// `components/organic/tokens.ts`; this file must stay CommonJS, so they exist twice.
const neutral = {
  100: "#f9f4ed",
  200: "#eee7db",
  300: "#dcd3c4",
  400: "#c0b6a5",
  500: "#a19786",
  600: "#82796a",
  700: "#645c50",
  800: "#474238",
  900: "#2e2b25",
};

const accent = {
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
};

const accent2 = {
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
};

/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./app/**/*.{ts,tsx}", "./components/**/*.{ts,tsx}", "../../packages/ui/src/**/*.{ts,tsx}"],
  presets: [require("nativewind/preset"), require("@fm/config/tailwind.preset.cjs")],
  theme: {
    extend: {
      colors: {
        neutral,
        accent,
        accent2,
        // The shared roles @fm/ui renders against, repainted in Organic. Doing it here
        // (rather than in the preset) is what keeps Button/Card/Field on-brand in this app
        // without touching apps/finance.
        primary: { DEFAULT: accent.DEFAULT, fg: "#ffffff", muted: accent[200] },
        bg: "#f5ead8",
        surface: neutral[100],
        border: neutral[300],
        divider: "rgba(32,30,29,0.16)",
        fg: "#201e1d",
        muted: neutral[600],
        // Organic is a light, warm system with no dark counterpart in the design. The dark
        // roles stay defined so shared @fm/ui `dark:` classes resolve to something warm
        // instead of the finance app's blue-greys.
        "bg-dark": accent[900],
        "surface-dark": "#2e2b25",
        "border-dark": neutral[800],
        "fg-dark": accent[100],
        "muted-dark": neutral[400],
        expense: "#a5341f",
        error: "#a5341f",
      },
      borderRadius: {
        xl: "20px",
        "2xl": "26px",
        "3xl": "36px",
      },
      fontFamily: {
        // Named `cap`/`fig` rather than `heading`/`bold` so they can never collide with
        // Tailwind's own font-weight utilities — with per-weight TTFs, `font-bold` alone
        // would not pick the right file. Faces are Alegreya (display) and Nunito Sans (body);
        // the utility keys keep their original names so no screen has to change.
        cap: ["Alegreya_800ExtraBold"],
        fig: ["NunitoSans_400Regular"],
        "fig-med": ["NunitoSans_500Medium"],
        "fig-semi": ["NunitoSans_600SemiBold"],
        "fig-bold": ["NunitoSans_700Bold"],
        "fig-x": ["NunitoSans_800ExtraBold"],
      },
    },
  },
};
