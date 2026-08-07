import assert from "node:assert/strict";
import { test } from "node:test";

import { normalizeFilters, queryKeys } from "./queryKeys.ts";

const march = { from: "2026-03-01", to: "2026-03-31" };

test("every finance key starts with the finance domain", () => {
  const keys = [
    queryKeys.accounts(),
    queryKeys.account("a1"),
    queryKeys.categories(),
    queryKeys.transactions({ range: march }),
    queryKeys.summary(march),
    queryKeys.breakdown(march, 1),
  ];
  for (const key of keys) {
    assert.equal(key[0], "finance", `key ${JSON.stringify(key)} is not domain-prefixed`);
  }
});

test("account id filters do not depend on order", () => {
  assert.deepEqual(queryKeys.summary(march, ["b", "a"]), queryKeys.summary(march, ["a", "b"]));
  assert.deepEqual(
    queryKeys.breakdown(march, 1, ["b", "a"]),
    queryKeys.breakdown(march, 1, ["a", "b"]),
  );
});

test("normalizeFilters collapses undefined and empty to the same shape", () => {
  assert.deepEqual(normalizeFilters({ range: march }), {
    range: march,
    accountIds: [],
    categoryIds: [],
    type: 0,
    search: "",
  });
  assert.deepEqual(
    normalizeFilters({ range: march, accountIds: [], search: "  ", type: 0 }),
    normalizeFilters({ range: march }),
  );
});

test("normalizeFilters sorts ids and trims the search term", () => {
  assert.deepEqual(normalizeFilters({ range: march, accountIds: ["b", "a"], search: " food " }), {
    range: march,
    accountIds: ["a", "b"],
    categoryIds: [],
    type: 0,
    search: "food",
  });
});

test("different periods never share a key", () => {
  const april = { from: "2026-04-01", to: "2026-04-30" };
  assert.notDeepEqual(queryKeys.summary(march), queryKeys.summary(april));
});
