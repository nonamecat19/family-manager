import type { PluralForms } from "./plural.ts";

/** A leaf is either a plain (optionally interpolated, `{like_this}`) string, or a set of
 * plural forms selected by a `count` param. Every screen's copy bottoms out in one of these
 * two shapes — never a nested object — so the key set is a flat, easily-diffed dictionary. */
export type Leaf = string | PluralForms;

export type Translations = Record<string, Leaf>;

/** The subset of a dictionary's keys whose value is a plain string. `bootT` is restricted to
 * these so it cannot grow into a second, provider-free `t()`. */
export type StaticKey<D extends Translations> = {
  [K in keyof D]: D[K] extends string ? K : never;
}[keyof D];
