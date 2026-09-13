import { organicTheme } from "@fm/theme";

export const organic = {
  bg: organicTheme.bg,
  surface: organicTheme.surface,
  text: organicTheme.text,
  divider: organicTheme.divider,
  muted: organicTheme.muted,
  neutral: organicTheme.neutral,
  accent: organicTheme.accent,
  accent2: organicTheme.accent2,
  danger: organicTheme.danger,
} as const;

const CATEGORY_TINTS = [
  { bg: organic.accent2[200], fg: organic.accent2[800] },
  { bg: organic.accent[200], fg: organic.accent[800] },
  { bg: organic.accent2[300], fg: organic.accent2[900] },
  { bg: organic.accent[300], fg: organic.accent[900] },
] as const;

export interface Tint {
  bg: string;
  fg: string;
}

const NAMED_TINTS: Record<string, number> = {
  breakfast: 0,
  lunch: 1,
  dinner: 2,
  dessert: 3,
};

export function tintFor(name: string | undefined, fallbackIndex = 0): Tint {
  const key = (name ?? "").trim().toLowerCase();
  const named = NAMED_TINTS[key];
  if (named !== undefined) return CATEGORY_TINTS[named]!;
  const hash = key === "" ? fallbackIndex : hashString(key);
  return CATEGORY_TINTS[hash % CATEGORY_TINTS.length]!;
}

function hashString(s: string): number {
  let h = 0;
  for (let i = 0; i < s.length; i++) h = (h * 31 + s.charCodeAt(i)) >>> 0;
  return h;
}

export function initialOf(text: string): string {
  const first = text.trim()[0];
  return first ? first.toUpperCase() : "?";
}
