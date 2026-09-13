import type { PluralForms } from "./plural.ts";

export type Leaf = string | PluralForms;

export type Translations = Record<string, Leaf>;

export type StaticKey<D extends Translations> = {
  [K in keyof D]: D[K] extends string ? K : never;
}[keyof D];
