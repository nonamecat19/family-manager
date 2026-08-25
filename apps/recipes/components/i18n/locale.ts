import * as Localization from "expo-localization";

export const LOCALES = ["en", "uk"] as const;
export type Locale = (typeof LOCALES)[number];

export const DEFAULT_LOCALE: Locale = "en";

/** Any `uk`/`uk-UA`/... device tag maps to Ukrainian; everything else falls back to English —
 * the cookbook ships two languages, not "whatever the phone happens to be set to". */
export function localeFromTag(tag: string | null | undefined): Locale {
  if (!tag) return DEFAULT_LOCALE;
  return tag.toLowerCase().startsWith("uk") ? "uk" : DEFAULT_LOCALE;
}

/** The device's best-guess locale, read once at startup. */
export function deviceLocale(): Locale {
  const [primary] = Localization.getLocales();
  return localeFromTag(primary?.languageTag ?? primary?.languageCode ?? null);
}
