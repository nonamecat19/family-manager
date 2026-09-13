import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import * as SecureStore from "expo-secure-store";
import { View } from "react-native";

import { deviceLocale } from "./locale.ts";
import { interpolate, selectPlural } from "./plural.ts";
import type { Leaf, StaticKey, Translations } from "./schema.ts";

export interface I18nConfig<L extends string, D extends Translations> {
  locales: readonly L[];
  defaultLocale: L;
  storageKey: string;
  dictionaries: Record<L, Record<keyof D, Leaf>>;
  bootBackground: string;
}

export interface I18n<L extends string, D extends Translations> {
  I18nProvider: (props: { children: ReactNode }) => ReactNode;
  useI18n: () => I18nContextValue<L, D>;
  bootT: (key: StaticKey<D>) => string;
  deviceLocale: () => L;
}

export interface I18nContextValue<L extends string, D extends Translations> {
  locale: L;
  setLocale: (locale: L) => void;
  t: (key: keyof D, params?: Record<string, string | number>) => string;
}

export function createI18n<L extends string, D extends Translations>(
  config: I18nConfig<L, D>,
): I18n<L, D> {
  const { locales, defaultLocale, storageKey, dictionaries, bootBackground } = config;

  const resolveDeviceLocale = (): L => deviceLocale(locales, defaultLocale);

  const I18nContext = createContext<I18nContextValue<L, D> | null>(null);

  function I18nProvider({ children }: { children: ReactNode }) {
    const [locale, setLocaleState] = useState<L>(resolveDeviceLocale);
    const [ready, setReady] = useState(false);

    useEffect(() => {
      let cancelled = false;
      SecureStore.getItemAsync(storageKey)
        .then((stored) => {
          if (cancelled) return;
          if (stored && (locales as readonly string[]).includes(stored)) {
            setLocaleState(stored as L);
          }
        })
        .catch(() => {
        })
        .finally(() => {
          if (!cancelled) setReady(true);
        });
      return () => {
        cancelled = true;
      };
    }, []);

    const setLocale = useCallback((next: L) => {
      setLocaleState(next);
      SecureStore.setItemAsync(storageKey, next).catch(() => {
      });
    }, []);

    const t = useCallback(
      (key: keyof D, params?: Record<string, string | number>): string => {
        const leaf = dictionaries[locale][key];
        if (typeof leaf === "string") return interpolate(leaf, params);
        if (leaf === undefined) return String(key);
        const count = params?.count;
        if (typeof count !== "number") {
          throw new Error(
            `i18n: "${String(key)}" is a pluralized key and needs a numeric "count" param`,
          );
        }
        return selectPlural(locale, count, leaf, params);
      },
      [locale],
    );

    const value = useMemo<I18nContextValue<L, D>>(
      () => ({ locale, setLocale, t }),
      [locale, setLocale, t],
    );

    if (!ready) return <View style={{ flex: 1, backgroundColor: bootBackground }} />;

    return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
  }

  function useI18n(): I18nContextValue<L, D> {
    const ctx = useContext(I18nContext);
    if (!ctx) throw new Error("useI18n must be used inside an I18nProvider");
    return ctx;
  }

  function bootT(key: StaticKey<D>): string {
    const leaf = dictionaries[resolveDeviceLocale()][key];
    return typeof leaf === "string" ? leaf : String(key);
  }

  return { I18nProvider, useI18n, bootT, deviceLocale: resolveDeviceLocale };
}
