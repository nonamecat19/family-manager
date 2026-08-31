import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import * as SecureStore from "expo-secure-store";
import { View } from "react-native";

import { deviceLocale, LOCALES, type Locale } from "./locale.ts";
import { interpolate, selectPlural } from "./plural.ts";
import type { Leaf } from "./translations/schema.ts";
import { en } from "./translations/en.ts";
import { uk } from "./translations/uk.ts";
import { organic } from "../organic/tokens.ts";

export type TranslationKey = keyof typeof en;

/** The subset of keys whose `en` value is a plain string (not `PluralForms`) — what `bootT`
 * below is restricted to. */
export type StaticTranslationKey = {
  [K in TranslationKey]: (typeof en)[K] extends string ? K : never;
}[TranslationKey];

const DICTS: Record<Locale, Record<TranslationKey, Leaf>> = { en, uk };

/** App-local, like the rest of i18n — `secureTokenStore` in @fm/auth owns `fm.session`, this
 * key is namespaced under the app so another app's language choice never collides. */
const STORAGE_KEY = "fm.recipes.locale";

interface I18nContextValue {
  locale: Locale;
  setLocale: (locale: Locale) => void;
  /** Looks up `key` in the active locale's dictionary. Plain strings are interpolated with
   * `params` (`{name}` → `params.name`); plural entries additionally require a numeric
   * `params.count` to pick the CLDR form. */
  t: (key: TranslationKey, params?: Record<string, string | number>) => string;
}

const I18nContext = createContext<I18nContextValue | null>(null);

export function I18nProvider({ children }: { children: ReactNode }) {
  // The device's own locale is the fallback; the persisted override (if any) is read before
  // the first paint below, so a Ukrainian phone whose owner picked English never flashes
  // Ukrainian first.
  const [locale, setLocaleState] = useState<Locale>(deviceLocale());
  const [ready, setReady] = useState(false);

  useEffect(() => {
    let cancelled = false;
    SecureStore.getItemAsync(STORAGE_KEY)
      .then((stored) => {
        if (cancelled) return;
        if (stored && (LOCALES as readonly string[]).includes(stored)) setLocaleState(stored as Locale);
      })
      .catch(() => {
        // A keystore read failure just means "no saved preference" — the device locale
        // already in state is a fine fallback.
      })
      .finally(() => {
        if (!cancelled) setReady(true);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const setLocale = useCallback((next: Locale) => {
    setLocaleState(next);
    SecureStore.setItemAsync(STORAGE_KEY, next).catch(() => {
      // Best-effort persistence: the choice still applies for the rest of this session even
      // if the keystore write failed, rather than surfacing an unhandled rejection.
    });
  }, []);

  const t = useCallback(
    (key: TranslationKey, params?: Record<string, string | number>): string => {
      const leaf = DICTS[locale][key];
      if (typeof leaf === "string") return interpolate(leaf, params);
      const count = params?.count;
      if (typeof count !== "number") {
        throw new Error(`i18n: "${key}" is a pluralized key and needs a numeric "count" param`);
      }
      return selectPlural(locale, count, leaf, params);
    },
    [locale],
  );

  const value = useMemo<I18nContextValue>(() => ({ locale, setLocale, t }), [locale, setLocale, t]);

  // Hold every consumer behind the persisted-locale read: without this, a saved choice that
  // differs from the device locale renders wrong for one frame and then flips. The read is a
  // handful of milliseconds, so a bare ground-coloured view (no text to pick a language for
  // yet) is all this needs — the screen underneath renders in its final locale the instant
  // this resolves.
  if (!ready) return <View style={{ flex: 1, backgroundColor: organic.bg }} />;

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function useI18n(): I18nContextValue {
  const ctx = useContext(I18nContext);
  if (!ctx) throw new Error("useI18n must be used inside an I18nProvider");
  return ctx;
}

/**
 * A synchronous, provider-free lookup for the sliver of app boot that renders before
 * `I18nProvider` can mount at all — root `app/_layout.tsx`'s font-loading and
 * session-restoring labels, both shown ahead of any `(app)`/`(auth)` layout.
 *
 * Resolves against the DEVICE locale only, never the persisted choice: `SecureStore` is
 * async, and these labels render before that read could possibly resolve. A user whose
 * saved preference differs from their phone's language will see these two boot labels in
 * the phone's language for a fraction of a second, until `I18nProvider` mounts downstream
 * and switches everything to their saved choice — an accepted, intentional gap, not an
 * oversight.
 *
 * Restricted to `StaticTranslationKey` (plain strings only) so it can't grow into a second,
 * provider-free `t()` — interpolation and pluralization stay behind `useI18n().t()`.
 * Nothing outside the pre-provider boot path should call this.
 */
export function bootT(key: StaticTranslationKey): string {
  return DICTS[deviceLocale()][key] as string;
}
