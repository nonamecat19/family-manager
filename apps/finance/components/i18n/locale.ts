import * as Localization from "expo-localization";

export const LOCALES = ["uk", "en"] as const;
export type Locale = (typeof LOCALES)[number];

/**
 * Ukrainian, not English. Every one of the eleven designed screens is drawn in Ukrainian and
 * the household this app was designed for speaks it; English exists as a fallback for a
 * device that is not set to Ukrainian, not as the primary face of the app. (apps/recipes
 * defaults the other way — that is a per-app choice, not a repo convention.)
 */
export const DEFAULT_LOCALE: Locale = "uk";

/** Any `en`/`en-GB`/... device tag maps to English; everything else — including the Ukrainian
 * this app is written in — falls back to the default. Two languages ship, not "whatever the
 * phone happens to be set to". */
export function localeFromTag(tag: string | null | undefined): Locale {
  if (!tag) return DEFAULT_LOCALE;
  return tag.toLowerCase().startsWith("en") ? "en" : DEFAULT_LOCALE;
}

/** The device's best-guess locale, read once at startup. */
export function deviceLocale(): Locale {
  const [primary] = Localization.getLocales();
  return localeFromTag(primary?.languageTag ?? primary?.languageCode ?? null);
}
