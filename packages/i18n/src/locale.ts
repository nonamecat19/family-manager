import * as Localization from "expo-localization";

import { localeFromTag } from "./resolve.ts";

/** The device's best-guess locale, narrowed to the shipped set. */
export function deviceLocale<L extends string>(locales: readonly L[], fallback: L): L {
  const [primary] = Localization.getLocales();
  return localeFromTag(primary?.languageTag ?? primary?.languageCode ?? null, locales, fallback);
}
