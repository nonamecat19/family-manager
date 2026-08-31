import type { Locale } from "./locale.ts";

/**
 * One template per CLDR plural category. `other` is the only one every locale must define —
 * Ukrainian additionally distinguishes `one`/`few`/`many` (1, 2-4, 5+ with the usual
 * mod-10/mod-100 exceptions).
 */
export interface PluralForms {
  one?: string;
  few?: string;
  many?: string;
  other: string;
}

type PluralCategory = keyof PluralForms;

/**
 * Hand-written CLDR category selection — no `Intl.PluralRules`. Hermes (React Native's
 * default JS engine, and this app sets no `jsEngine` override) does not implement it and the
 * app ships no polyfill, so calling it here would throw on every device launch while still
 * looking fine in `tsc`/web, where the host JS engine (V8/JSC) happens to have it.
 *
 * Only the two locales this app ships need a rule; `count` is always treated as a
 * non-negative integer, which is all a UI count ever is.
 */
function categoryFor(locale: Locale, count: number): PluralCategory {
  const n = Math.abs(Math.trunc(count));
  if (locale === "uk") {
    const mod10 = n % 10;
    const mod100 = n % 100;
    if (mod10 === 1 && mod100 !== 11) return "one";
    if (mod10 >= 2 && mod10 <= 4 && !(mod100 >= 12 && mod100 <= 14)) return "few";
    return "many";
  }
  // en (CLDR): singular only for exactly 1.
  return n === 1 ? "one" : "other";
}

/** Picks the right form for `count` under `locale`'s plural rules and substitutes `{count}`
 * (plus anything in `params`) into it. */
export function selectPlural(
  locale: Locale,
  count: number,
  forms: PluralForms,
  params?: Record<string, string | number>,
): string {
  const category = categoryFor(locale, count);
  const template = forms[category] ?? forms.other;
  return interpolate(template, { count, ...params });
}

export function interpolate(template: string, params?: Record<string, string | number>): string {
  if (!params) return template;
  return template.replace(/\{(\w+)\}/g, (match, name: string) =>
    Object.hasOwn(params, name) ? String(params[name]) : match,
  );
}
