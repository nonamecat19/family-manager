// Nocturne — the finance app's own palette, taken verbatim from the design canvas
// `docs/design/finance/Family Money Manager.dc.html` and its Nocturne stylesheet.
//
// It overrides the shared @fm/config preset's roles here rather than in the preset itself:
// the preset is shared with apps/recipes (`just impact pkg:@fm/config`), and repainting it
// would repaint that app too. Values mirror `components/nocturne/tokens.ts` — this file must
// stay CommonJS, which is why the palette exists twice. Change one, change the other.
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

// Nocturne carries no danger role. The design adds exactly one value for it — a budget breach
// is the single thing in this app that has to read at a glance — generated in OKLCH at
// accent-400's own lightness and chroma (L 0.734, C 0.125, hue 27).
const overspend = "#e5928a";

const bg = "#161826";
const surface = "#232532";

/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./app/**/*.{ts,tsx}", "./components/**/*.{ts,tsx}", "../../packages/ui/src/**/*.{ts,tsx}"],
  presets: [require("nativewind/preset"), require("@fm/config/tailwind.preset.cjs")],
  theme: {
    extend: {
      colors: {
        neutral,
        accent,
        overspend,
        // The ground the canvas itself sits on, behind the phone frame.
        canvas: "#101120",
        // The shared roles @fm/ui renders against, repainted in Nocturne. Nocturne is
        // dark-only, so the `*-dark` counterparts resolve to the same values — a shared
        // component using `dark:` classes must not fall back to the preset's blue-greys.
        primary: { DEFAULT: accent.DEFAULT, fg: "#161826", muted: accent[900] },
        bg,
        surface,
        border: neutral[800],
        divider: "rgba(233,233,237,.16)",
        fg: "#e9e9ed",
        muted: neutral[500],
        "bg-dark": bg,
        "surface-dark": surface,
        "border-dark": neutral[800],
        "fg-dark": "#e9e9ed",
        "muted-dark": neutral[500],
        error: overspend,
        expense: overspend,
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
      borderRadius: {
        sm: "4px",
        md: "8px",
        lg: "14px",
      },
    },
  },
};
