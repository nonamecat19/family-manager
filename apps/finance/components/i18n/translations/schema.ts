import type { PluralForms } from "../plural.ts";

/** A leaf is either a plain (optionally interpolated, `{like_this}`) string, or a set of
 * plural forms selected by a `count` param. Every screen's copy bottoms out in one of these
 * two shapes — never a nested object — so the key set is a flat, easily-diffed dictionary. */
export type Leaf = string | PluralForms;

export type Translations = Record<string, Leaf>;
