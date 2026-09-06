import assert from "node:assert/strict";
import { test } from "node:test";

import { interpolate, pluralCategory, pluralRuleFor, selectPlural } from "./plural.ts";

test("english is singular only for exactly one", () => {
  assert.equal(pluralCategory("en", 0), "other");
  assert.equal(pluralCategory("en", 1), "one");
  assert.equal(pluralCategory("en", 2), "other");
  assert.equal(pluralCategory("en", 21), "other");
});

test("ukrainian follows the CLDR one/few/many bands", () => {
  // one: mod10 == 1, except the 11 band.
  for (const n of [1, 21, 101, 1031]) assert.equal(pluralCategory("uk", n), "one", `n=${n}`);
  assert.equal(pluralCategory("uk", 11), "many");
  assert.equal(pluralCategory("uk", 111), "many");

  // few: mod10 in 2..4, except the 12-14 band.
  for (const n of [2, 3, 4, 22, 34, 104]) assert.equal(pluralCategory("uk", n), "few", `n=${n}`);
  for (const n of [12, 13, 14, 112, 114]) assert.equal(pluralCategory("uk", n), "many", `n=${n}`);

  // many: everything else, zero included.
  for (const n of [0, 5, 9, 10, 25, 100]) assert.equal(pluralCategory("uk", n), "many", `n=${n}`);
});

test("a region subtag resolves to its language's rule", () => {
  assert.equal(pluralCategory("uk-UA", 2), "few");
  assert.equal(pluralCategory("en-GB", 1), "one");
  assert.equal(pluralCategory("en_US", 3), "other");
});

test("an unregistered language falls back to the english rule, not to a throw", () => {
  assert.equal(pluralRuleFor("de")(1), "one");
  assert.equal(pluralRuleFor("de")(7), "other");
});

test("counts are normalised to non-negative integers", () => {
  assert.equal(pluralCategory("uk", -2), "few");
  assert.equal(pluralCategory("en", 1.7), "one");
});

test("selectPlural picks the form and substitutes count", () => {
  const forms = { one: "{count} нотатка", few: "{count} нотатки", many: "{count} нотаток", other: "{count} нотаток" };
  assert.equal(selectPlural("uk", 1, forms), "1 нотатка");
  assert.equal(selectPlural("uk", 3, forms), "3 нотатки");
  assert.equal(selectPlural("uk", 7, forms), "7 нотаток");
});

test("selectPlural falls back to `other` when the chosen form is absent", () => {
  // A dictionary that only defines `other` is legal — `other` is the one required form.
  assert.equal(selectPlural("uk", 1, { other: "{count} items" }), "1 items");
});

test("selectPlural merges extra params alongside count", () => {
  const forms = { one: "{count} note in {book}", other: "{count} notes in {book}" };
  assert.equal(selectPlural("en", 2, forms, { book: "Ideas" }), "2 notes in Ideas");
});

test("interpolate substitutes only known params and leaves the rest verbatim", () => {
  assert.equal(interpolate("hi {name}", { name: "Olena" }), "hi Olena");
  assert.equal(interpolate("hi {name}", {}), "hi {name}");
  assert.equal(interpolate("hi {name}"), "hi {name}");
  assert.equal(interpolate("{a} and {b}", { a: 1, b: 2 }), "1 and 2");
});

test("interpolate does not recurse into a substituted value", () => {
  // A translated string that happens to contain braces must not become a template itself.
  assert.equal(interpolate("{a}", { a: "{b}", b: "boom" }), "{b}");
});
