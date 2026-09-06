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
  /** Every locale the app ships, most-specific first. */
  locales: readonly L[];
  /** Where a device tag that matches nothing lands. */
  defaultLocale: L;
  /**
   * The persisted-choice key. App-namespaced (`fm.<app>.locale`) so one app's language choice
   * never collides with another's — `@fm/auth`'s `secureTokenStore` owns `fm.session`, and
   * this sits beside it under the app's own prefix.
   */
  storageKey: string;
  /**
   * One dictionary per shipped locale. `D` is the SOURCE dictionary — the `en.ts` object
   * literal, with its string literal types intact, which is what lets `bootT` narrow to the
   * non-plural keys. Every other locale only has to match that key set with `Leaf` values,
   * which is the discipline the translation files already follow (`uk.ts` is typed against
   * `keyof typeof en`).
   */
  dictionaries: Record<L, Record<keyof D, Leaf>>;
  /**
   * The ground painted while the persisted choice is read. It is a bare colour rather than a
   * component because there is no text to pick a language for yet — see `I18nProvider` below.
   */
  bootBackground: string;
}

export interface I18n<L extends string, D extends Translations> {
  I18nProvider: (props: { children: ReactNode }) => ReactNode;
  useI18n: () => I18nContextValue<L, D>;
  bootT: (key: StaticKey<D>) => string;
  /** The device locale narrowed to this app's shipped set — the value the provider starts on. */
  deviceLocale: () => L;
}

export interface I18nContextValue<L extends string, D extends Translations> {
  locale: L;
  setLocale: (locale: L) => void;
  /** Looks up `key` in the active locale's dictionary. Plain strings are interpolated with
   * `params` (`{name}` → `params.name`); plural entries additionally require a numeric
   * `params.count` to pick the CLDR form. */
  t: (key: keyof D, params?: Record<string, string | number>) => string;
}

/**
 * Builds an app's i18n surface from its dictionaries. Everything that used to differ between
 * `apps/finance` and `apps/recipes` — the shipped locales, the default, the storage namespace,
 * the boot ground — is an argument here; the provider itself was byte-identical in both and now
 * exists once.
 */
export function createI18n<L extends string, D extends Translations>(
  config: I18nConfig<L, D>,
): I18n<L, D> {
  const { locales, defaultLocale, storageKey, dictionaries, bootBackground } = config;

  const resolveDeviceLocale = (): L => deviceLocale(locales, defaultLocale);

  const I18nContext = createContext<I18nContextValue<L, D> | null>(null);

  function I18nProvider({ children }: { children: ReactNode }) {
    // The device's own locale is the fallback; the persisted override (if any) is read before
    // the first paint below, so a Ukrainian phone whose owner picked English never flashes
    // Ukrainian first.
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

    const setLocale = useCallback((next: L) => {
      setLocaleState(next);
      SecureStore.setItemAsync(storageKey, next).catch(() => {
        // Best-effort persistence: the choice still applies for the rest of this session even
        // if the keystore write failed, rather than surfacing an unhandled rejection.
      });
    }, []);

    const t = useCallback(
      (key: keyof D, params?: Record<string, string | number>): string => {
        const leaf = dictionaries[locale][key];
        if (typeof leaf === "string") return interpolate(leaf, params);
        // `keyof D` makes this unreachable for a well-formed dictionary, but a dictionary is
        // data — a hand-edited translation file with a dropped line reaches here. Rendering
        // the key beats rendering "undefined" in the middle of a screen.
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

    // Hold every consumer behind the persisted-locale read: without this, a saved choice that
    // differs from the device locale renders wrong for one frame and then flips. The read is a
    // handful of milliseconds, so a bare ground-coloured view (no text to pick a language for
    // yet) is all this needs — the screen underneath renders in its final locale the instant
    // this resolves.
    if (!ready) return <View style={{ flex: 1, backgroundColor: bootBackground }} />;

    return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
  }

  function useI18n(): I18nContextValue<L, D> {
    const ctx = useContext(I18nContext);
    if (!ctx) throw new Error("useI18n must be used inside an I18nProvider");
    return ctx;
  }

  /**
   * A synchronous, provider-free lookup for the sliver of app boot that renders before
   * `I18nProvider` can mount at all — the root `app/_layout.tsx` labels shown ahead of any
   * `(app)`/`(auth)` layout.
   *
   * Resolves against the DEVICE locale only, never the persisted choice: `SecureStore` is
   * async, and these labels render before that read could possibly resolve. A user whose saved
   * preference differs from their phone's language sees these boot labels in the phone's
   * language for a fraction of a second, until `I18nProvider` mounts downstream and switches
   * everything to their saved choice — an accepted, intentional gap, not an oversight.
   *
   * Restricted to `StaticKey` (plain strings only) so it cannot grow into a second,
   * provider-free `t()` — interpolation and pluralization stay behind `useI18n().t()`.
   * Nothing outside the pre-provider boot path should call this.
   */
  function bootT(key: StaticKey<D>): string {
    const leaf = dictionaries[resolveDeviceLocale()][key];
    return typeof leaf === "string" ? leaf : String(key);
  }

  return { I18nProvider, useI18n, bootT, deviceLocale: resolveDeviceLocale };
}
