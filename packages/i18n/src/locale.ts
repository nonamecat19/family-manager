import * as Localization from "expo-localization";

import { localeFromTag } from "./resolve.ts";

export function deviceLocale<L extends string>(locales: readonly L[], fallback: L): L {
  const [primary] = Localization.getLocales();
  return localeFromTag(primary?.languageTag ?? primary?.languageCode ?? null, locales, fallback);
}
