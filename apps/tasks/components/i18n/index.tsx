import { useUpdateUserSettings, useUserSettings } from "@fm/api";
import { useAuth } from "@fm/auth";
import { createI18n } from "@fm/i18n";
import { useCallback, type ReactNode } from "react";

import { defaultUiTranslate, UiLabelsProvider, type UiTranslate } from "@fm/ui";

import { organic } from "../kit/tokens.ts";
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
  storageKey: "fm.tasks.locale",
  dictionaries: { en, uk },
  bootBackground: organic.bg,
});

const { I18nProvider: BaseI18nProvider, useI18n, bootT, deviceLocale } = i18n;

export { useI18n, bootT, deviceLocale };

function UiLabelsBridge({ children }: { children: ReactNode }) {
  const { t } = useI18n();
  const translate = useCallback<UiTranslate>(
    (key, vars) => (Object.hasOwn(en, key) ? t(key as TranslationKey, vars) : defaultUiTranslate(key, vars)),
    [t],
  );
  return <UiLabelsProvider translate={translate}>{children}</UiLabelsProvider>;
}

export function I18nProvider({ children }: { children: ReactNode }) {
  const { status } = useAuth();
  const authenticated = status === "authenticated";

  const settings = useUserSettings({ enabled: authenticated });
  const update = useUpdateUserSettings();

  const remoteLocale = settings.data?.settings?.locale ?? null;

  const onLocaleChange = useCallback(
    (next: string) => {
      if (!authenticated) return;
      update.mutate({ locale: next });
    },
    [authenticated, update],
  );

  return (
    <BaseI18nProvider remoteLocale={remoteLocale} onLocaleChange={onLocaleChange}>
      <UiLabelsBridge>{children}</UiLabelsBridge>
    </BaseI18nProvider>
  );
}

export { localeFromTag, interpolate, selectPlural, type PluralForms } from "@fm/i18n";