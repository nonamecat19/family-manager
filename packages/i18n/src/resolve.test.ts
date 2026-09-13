import assert from "node:assert/strict";
import { test } from "node:test";

import { localeFromTag } from "./resolve.ts";

const FINANCE = { locales: ["uk", "en"] as const, fallback: "uk" as const };
const RECIPES = { locales: ["en", "uk"] as const, fallback: "en" as const };

test("a shipped language matches whatever its region subtag is", () => {
  assert.equal(localeFromTag("en-GB", FINANCE.locales, FINANCE.fallback), "en");
  assert.equal(localeFromTag("uk-UA", FINANCE.locales, FINANCE.fallback), "uk");
  assert.equal(localeFromTag("uk-UA", RECIPES.locales, RECIPES.fallback), "uk");
  assert.equal(localeFromTag("en-US", RECIPES.locales, RECIPES.fallback), "en");
});

test("an unshipped language falls back to the app's own default, not the device's", () => {
  assert.equal(localeFromTag("de-DE", FINANCE.locales, FINANCE.fallback), "uk");
  assert.equal(localeFromTag("de-DE", RECIPES.locales, RECIPES.fallback), "en");
});

test("a missing tag falls back", () => {
  assert.equal(localeFromTag(null, FINANCE.locales, FINANCE.fallback), "uk");
  assert.equal(localeFromTag(undefined, RECIPES.locales, RECIPES.fallback), "en");
  assert.equal(localeFromTag("", FINANCE.locales, FINANCE.fallback), "uk");
});

test("matching is case-insensitive on both sides", () => {
  assert.equal(localeFromTag("EN-gb", FINANCE.locales, FINANCE.fallback), "en");
});

test("resolution does not depend on the order the app lists its locales", () => {
  for (const tag of ["en-GB", "uk-UA", "de"]) {
    const forward = localeFromTag(tag, ["uk", "en"] as const, "uk");
    const reversed = localeFromTag(tag, ["en", "uk"] as const, "uk");
    assert.equal(forward, reversed, tag);
  }
});

test("a more specific shipped locale wins over its bare language", () => {
  const locales = ["pt", "pt-br"] as const;
  assert.equal(localeFromTag("pt-BR", locales, "pt"), "pt-br");
  assert.equal(localeFromTag("pt-PT", locales, "pt"), "pt");
});
