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

export type PluralCategory = keyof PluralForms;

/**
 * A rule maps a non-negative integer to its CLDR category. Rules are registered per language
 * subtag rather than per app: the Ukrainian rule is the same Ukrainian rule everywhere, and an
 * app that ships a third language adds it here once instead of forking this module.
 */
export type PluralRule = (count: number) => PluralCategory;

/** English (CLDR): singular only for exactly 1. */
const english: PluralRule = (n) => (n === 1 ? "one" : "other");

/** Ukrainian (CLDR): one/few/many by mod-10 with the 11-14 exception band. */
const ukrainian: PluralRule = (n) => {
  const mod10 = n % 10;
  const mod100 = n % 100;
  if (mod10 === 1 && mod100 !== 11) return "one";
  if (mod10 >= 2 && mod10 <= 4 && !(mod100 >= 12 && mod100 <= 14)) return "few";
  return "many";
};

/**
 * Hand-written CLDR category selection — no `Intl.PluralRules`. Hermes (React Native's default
 * JS engine, and no app here sets a `jsEngine` override) does not implement it and the apps
 * ship no polyfill, so calling it would throw on every device launch while still looking fine
 * in `tsc` and on web, where the host engine (V8/JSC) happens to have it.
 */
const RULES: Record<string, PluralRule> = {
  en: english,
  uk: ukrainian,
};

/**
 * The rule for a locale, falling back to the English rule. The fallback is a shape, not a
 * translation: a locale with no registered rule still renders its own `other` form, it just
 * does not get a language-specific singular.
 */
export function pluralRuleFor(locale: string): PluralRule {
  const subtag = locale.toLowerCase().split(/[-_]/)[0] ?? "";
  return RULES[subtag] ?? english;
}

/** The category `count` falls into under `locale`. `count` is treated as a non-negative
 * integer, which is all a UI count ever is. */
export function pluralCategory(locale: string, count: number): PluralCategory {
  return pluralRuleFor(locale)(Math.abs(Math.trunc(count)));
}

/** Picks the right form for `count` under `locale`'s plural rules and substitutes `{count}`
 * (plus anything in `params`) into it. */
export function selectPlural(
  locale: string,
  count: number,
  forms: PluralForms,
  params?: Record<string, string | number>,
): string {
  const template = forms[pluralCategory(locale, count)] ?? forms.other;
  return interpolate(template, { count, ...params });
}

export function interpolate(template: string, params?: Record<string, string | number>): string {
  if (!params) return template;
  return template.replace(/\{(\w+)\}/g, (match, name: string) =>
    Object.hasOwn(params, name) ? String(params[name]) : match,
  );
}
