import { initialOf, nocturneCore, type Tint } from "@fm/theme";

export { initialOf, type Tint };

export const nocturne = {
  ...nocturneCore,

  rail: "#1b1d2c",
  list: "#191b29",

  accent2: {
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
  },
} as const;

export const FONT_NOTE = "Inter; weights 400/500/600/700" as const;

const MEMBER_TINTS: readonly Tint[] = [
  { bg: nocturne.accent[800], fg: nocturne.accent[200] },
  { bg: nocturne.accent2[700], fg: nocturne.accent2[100] },
  { bg: nocturne.accent[700], fg: nocturne.accent[100] },
  { bg: nocturne.accent2[800], fg: nocturne.accent2[200] },
  { bg: nocturne.neutral[800], fg: nocturne.neutral[200] },
];

export function tintFor(name: string | undefined, fallbackIndex = 0): Tint {
  const key = (name ?? "").trim().toLowerCase();
  const slot = key === "" ? fallbackIndex : hashString(key);
  return MEMBER_TINTS[slot % MEMBER_TINTS.length]!;
}

function hashString(s: string): number {
  let h = 0;
  for (let i = 0; i < s.length; i++) h = (h * 31 + s.charCodeAt(i)) >>> 0;
  return h;
}

