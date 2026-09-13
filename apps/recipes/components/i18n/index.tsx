import { createI18n } from "@fm/i18n";

import { organic } from "../organic/tokens.ts";
import { en } from "./translations/en.ts";
import { uk } from "./translations/uk.ts";

export const LOCALES = ["en", "uk"] as const;
export type Locale = (typeof LOCALES)[number];
export const DEFAULT_LOCALE: Locale = "en";

export type TranslationKey = keyof typeof en;

export type StaticTranslationKey = {
  [K in TranslationKey]: (typeof en)[K] extends string ? K : never;
}[TranslationKey];

const i18n = createI18n<Locale, typeof en>({
  locales: LOCALES,
  defaultLocale: DEFAULT_LOCALE,
  storageKey: "fm.recipes.locale",
  dictionaries: { en, uk },
  bootBackground: organic.bg,
});

export const { I18nProvider, useI18n, bootT, deviceLocale } = i18n;

export { localeFromTag, type PluralForms } from "@fm/i18n";
