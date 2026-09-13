import { createI18n } from "@fm/i18n";

import { nocturne } from "../nocturne/tokens.ts";
import { en } from "./translations/en.ts";
import { uk } from "./translations/uk.ts";

export const LOCALES = ["uk", "en"] as const;
export type Locale = (typeof LOCALES)[number];
export const DEFAULT_LOCALE: Locale = "uk";

export type TranslationKey = keyof typeof en;

export type StaticTranslationKey = {
  [K in TranslationKey]: (typeof en)[K] extends string ? K : never;
}[TranslationKey];

const i18n = createI18n<Locale, typeof en>({
  locales: LOCALES,
  defaultLocale: DEFAULT_LOCALE,
  storageKey: "fm.finance.locale",
  dictionaries: { en, uk },
  bootBackground: nocturne.bg,
});

export const { I18nProvider, useI18n, bootT, deviceLocale } = i18n;

export { localeFromTag, interpolate, selectPlural, type PluralForms } from "@fm/i18n";
