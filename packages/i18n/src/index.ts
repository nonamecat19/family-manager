export { createI18n, type I18n, type I18nConfig, type I18nContextValue } from "./createI18n.tsx";
export { deviceLocale } from "./locale.ts";
export { localeFromTag } from "./resolve.ts";
export {
  interpolate,
  pluralCategory,
  pluralRuleFor,
  selectPlural,
  type PluralCategory,
  type PluralForms,
  type PluralRule,
} from "./plural.ts";
export type { Leaf, StaticKey, Translations } from "./schema.ts";
