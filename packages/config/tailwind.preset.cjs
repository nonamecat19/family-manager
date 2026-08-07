// The Tailwind preset every app's tailwind.config.js extends. Token values live in
// @fm/theme (tokens.ts) and are mirrored here because Tailwind config is CommonJS and must
// stay loadable without a TS pipeline. packages/theme/src/tokens.test.ts asserts the two
// stay in sync — a drift there is a test failure, not a surprise at runtime.
const colors = {
  // Brand
  primary: {
    DEFAULT: "#2F855A",
    fg: "#FFFFFF",
    muted: "#C6F6D5",
  },
  // Semantic money colours: income is never red, expense is never green.
  income: "#2F855A",
  expense: "#C53030",
  transfer: "#2B6CB0",

  // Surfaces, light
  bg: "#F7FAFC",
  surface: "#FFFFFF",
  border: "#E2E8F0",
  fg: "#1A202C",
  muted: "#718096",

  // Surfaces, dark
  "bg-dark": "#12161C",
  "surface-dark": "#1A202C",
  "border-dark": "#2D3748",
  "fg-dark": "#F7FAFC",
  "muted-dark": "#A0AEC0",
};

const spacing = {
  xs: "4px",
  sm: "8px",
  md: "12px",
  lg: "16px",
  xl: "24px",
  "2xl": "32px",
};

const radius = {
  sm: "6px",
  md: "10px",
  lg: "16px",
  full: "9999px",
};

module.exports = {
  theme: {
    extend: {
      colors,
      spacing,
      borderRadius: radius,
      fontSize: {
        caption: ["12px", { lineHeight: "16px" }],
        body: ["15px", { lineHeight: "22px" }],
        title: ["20px", { lineHeight: "26px" }],
        display: ["32px", { lineHeight: "38px" }],
        // Balances and amounts get their own scale — they are the reason the app exists.
        amount: ["28px", { lineHeight: "34px" }],
      },
    },
  },
};
