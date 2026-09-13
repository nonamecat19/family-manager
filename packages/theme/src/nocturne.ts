export const nocturneCore = {
  bg: "#161826",
  surface: "#232532",
  text: "#e9e9ed",
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

  radius: { sm: 4, md: 8, lg: 14 },
} as const;

export interface Tint {
  bg: string;
  fg: string;
}

export function initialOf(text: string): string {
  const first = text.trim()[0];
  return first ? first.toUpperCase() : "?";
}
