export interface PluralForms {
  one?: string;
  few?: string;
  many?: string;
  other: string;
}

export type PluralCategory = keyof PluralForms;

export type PluralRule = (count: number) => PluralCategory;

const english: PluralRule = (n) => (n === 1 ? "one" : "other");

const ukrainian: PluralRule = (n) => {
  const mod10 = n % 10;
  const mod100 = n % 100;
  if (mod10 === 1 && mod100 !== 11) return "one";
  if (mod10 >= 2 && mod10 <= 4 && !(mod100 >= 12 && mod100 <= 14)) return "few";
  return "many";
};

const RULES: Record<string, PluralRule> = {
  en: english,
  uk: ukrainian,
};

export function pluralRuleFor(locale: string): PluralRule {
  const subtag = locale.toLowerCase().split(/[-_]/)[0] ?? "";
  return RULES[subtag] ?? english;
}

export function pluralCategory(locale: string, count: number): PluralCategory {
  return pluralRuleFor(locale)(Math.abs(Math.trunc(count)));
}

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
