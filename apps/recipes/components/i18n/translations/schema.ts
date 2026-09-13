import type { PluralForms } from "@fm/i18n";

export type Leaf = string | PluralForms;

export type Translations = Record<string, Leaf>;
