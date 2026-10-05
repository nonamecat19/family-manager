import { useUpdateUserSettings, useUserSettings } from "@fm/api";
import { useAuth } from "@fm/auth";
import { createI18n } from "@fm/i18n";
import { useCallback, type ReactNode } from "react";
import { organic, UiLabelsProvider, type UiTranslate } from "@fm/ui";

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

const { I18nProvider: BaseI18nProvider, useI18n, bootT, deviceLocale } = i18n;

export { useI18n, bootT, deviceLocale };

function UiLabelsBridge({ children }: { children: ReactNode }) {
  const { t } = useI18n();
  const translate = useCallback<UiTranslate>(
    (key, vars) => t(key as TranslationKey, vars),
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

export { localeFromTag, type PluralForms } from "@fm/i18n";
