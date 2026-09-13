import { colors } from "./tokens.ts";

export interface Scheme {
  bg: string;
  surface: string;
  border: string;
  fg: string;
  muted: string;
  primary: string;
  primaryFg: string;
  income: string;
  expense: string;
  transfer: string;
}

export const lightScheme: Scheme = {
  bg: colors.bg,
  surface: colors.surface,
  border: colors.border,
  fg: colors.fg,
  muted: colors.muted,
  primary: colors.primary,
  primaryFg: colors.primaryFg,
  income: colors.income,
  expense: colors.expense,
  transfer: colors.transfer,
};

export const darkScheme: Scheme = {
  bg: colors.bgDark,
  surface: colors.surfaceDark,
  border: colors.borderDark,
  fg: colors.fgDark,
  muted: colors.mutedDark,
  primary: colors.primary,
  primaryFg: colors.primaryFg,
  income: colors.income,
  expense: colors.expense,
  transfer: colors.transfer,
};

export type ColorSchemeName = "light" | "dark";

export function schemeFor(name: ColorSchemeName | null | undefined): Scheme {
  return name === "dark" ? darkScheme : lightScheme;
}

export function categoryColor(
  ownColor: string | undefined,
  index: number,
  palette: readonly string[],
): string {
  if (ownColor && ownColor.trim() !== "") return ownColor;
  const slot = palette[index % palette.length];
  return slot ?? colors.muted;
}
