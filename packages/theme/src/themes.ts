import type { Theme } from "./theme.ts";

export const organicTheme: Theme = {
  name: "organic",
  scheme: "light",

  bg: "#f5ead8",
  surface: "#f9f4ed",
  text: "#201e1d",
  muted: "#82796a",
  divider: "rgba(32,30,29,0.16)",

  neutral: {
    100: "#f9f4ed",
    200: "#eee7db",
    300: "#dcd3c4",
    400: "#c0b6a5",
    500: "#a19786",
    600: "#82796a",
    700: "#645c50",
    800: "#474238",
    900: "#2e2b25",
  },

  accent: {
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
  },

  accent2: {
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
  },

  danger: "#a5341f",
  accentFg: "#ffffff",
  dangerFg: "#ffffff",

  radius: { sm: 20, md: 26, lg: 36 },
  fieldStyle: "filled",
  labelCase: "sentence",
};

export const themes = { organic: organicTheme } as const;

export type ThemeName = keyof typeof themes;
