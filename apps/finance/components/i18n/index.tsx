/**
 * The app's i18n surface. The provider, the CLDR plural rules and the locale resolver live in
 * `@fm/i18n` — this module is the configuration that used to be tangled up with them: which
 * locales ship, which one a non-matching device gets, where the choice is persisted, and the
 * ground painted while that read resolves.
 */
import { createI18n } from "@fm/i18n";

import { nocturne } from "../nocturne/tokens.ts";
import { en } from "./translations/en.ts";
import { uk } from "./translations/uk.ts";

/**
 * Ukrainian, not English. Every one of the eleven designed screens is drawn in Ukrainian and
 * the household this app was designed for speaks it; English exists as a fallback for a
 * device that is not set to Ukrainian, not as the primary face of the app. (apps/recipes
 * defaults the other way — that is a per-app choice, not a repo convention.)
 */
export const LOCALES = ["uk", "en"] as const;
export type Locale = (typeof LOCALES)[number];
export const DEFAULT_LOCALE: Locale = "uk";

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
  storageKey: "fm.finance.locale",
  dictionaries: { en, uk },
  bootBackground: nocturne.bg,
});

export const { I18nProvider, useI18n, bootT, deviceLocale } = i18n;

export { localeFromTag, interpolate, selectPlural, type PluralForms } from "@fm/i18n";
