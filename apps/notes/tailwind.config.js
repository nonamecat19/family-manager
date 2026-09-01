// Nocturne — the notes app's own palette. It deliberately overrides the shared
// @fm/config preset's brand/surface roles instead of editing the preset, because the preset
// is shared by every app that consumes it (see `just impact pkg:@fm/config`). Values mirror
// `components/nocturne/tokens.ts`; this file must stay CommonJS, so they exist twice.
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

const bg = "#161826";
const surface = "#232532";

// The two pane grounds the design uses that are not on the ramp: the sidebar sits a shade
// under the app ground, the note list a shade over it. Named rather than left as literals in
// a screen, because "the rail colour" is a role this app repeats on every screen.
const rail = "#1b1d2c";
const list = "#191b29";

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
        rail,
        list,
        // The shared roles @fm/ui renders against, repainted in Nocturne. Doing it here
        // (rather than in the preset) is what keeps Button/Card/Field on-brand in this app
        // without touching the shared preset.
        primary: { DEFAULT: accent.DEFAULT, fg: bg, muted: accent[900] },
        bg,
        surface,
        border: neutral[800],
        divider: "rgba(233,233,237,.16)",
        fg: "#e9e9ed",
        muted: neutral[500],
        // Nocturne is a dark-only system: there is no light pass of these screens, so the
        // `*-dark` counterparts resolve to the same values — a shared @fm/ui component using
        // `dark:` classes must not fall back to the preset's blue-greys.
        "bg-dark": bg,
        "surface-dark": surface,
        "border-dark": neutral[800],
        "fg-dark": "#e9e9ed",
        "muted-dark": neutral[500],
        expense: "#e5928a",
        error: "#e5928a",
      },
      borderRadius: {
        sm: "4px",
        md: "8px",
        lg: "14px",
      },
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
