/**
 * The app's i18n surface. The provider, the CLDR plural rules and the locale resolver live in
 * `@fm/i18n` — this module is the configuration that used to be tangled up with them: which
 * locales ship, which one a non-matching device gets, where the choice is persisted, and the
 * ground painted while that read resolves.
 */
import { createI18n } from "@fm/i18n";

import { organic } from "../organic/tokens.ts";
import { en } from "./translations/en.ts";
import { uk } from "./translations/uk.ts";

/** The cookbook ships two languages and leads with English; apps/finance defaults the other
 * way, which is a per-app choice rather than a repo convention. */
export const LOCALES = ["en", "uk"] as const;
export type Locale = (typeof LOCALES)[number];
export const DEFAULT_LOCALE: Locale = "en";

export type TranslationKey = keyof typeof en;

/** The subset of keys whose `en` value is a plain string (not `PluralForms`) — what `bootT`
 * is restricted to. */
export type StaticTranslationKey = {
  [K in TranslationKey]: (typeof en)[K] extends string ? K : never;
}[TranslationKey];

const i18n = createI18n<Locale, typeof en>({
  locales: LOCALES,
  defaultLocale: DEFAULT_LOCALE,
  // App-namespaced: `secureTokenStore` in @fm/auth owns `fm.session`, and this key sits
  // beside it under the app's own prefix so another app's language choice never collides.
  storageKey: "fm.recipes.locale",
  dictionaries: { en, uk },
  bootBackground: organic.bg,
});

export const { I18nProvider, useI18n, bootT, deviceLocale } = i18n;

export { localeFromTag, type PluralForms } from "@fm/i18n";
