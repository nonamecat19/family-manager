import assert from "node:assert/strict";
import { test } from "node:test";

import { normalizeTransactionFilters, queryKeys } from "./queryKeys.ts";
import { accountScope, familyScope, memberScope } from "./scope.ts";

const august = { granularity: "month", anchor: "2026-08-30" } as const;
const september = { granularity: "month", anchor: "2026-09-02" } as const;

test("every key starts with its domain segment", () => {
  const family = [queryKeys.familyDetail(), queryKeys.members(), queryKeys.invitations()];
  for (const key of family) {
    assert.equal(key[0], "family", `key ${JSON.stringify(key)} is not domain-prefixed`);
  }

  const recipes = [
    queryKeys.recipeCategories(),
    queryKeys.recipeSubcategories("c1"),
    queryKeys.recipesList({}),
    queryKeys.recipe("r1"),
    queryKeys.favoriteRecipes(),
    queryKeys.recipeComments("r1"),
    queryKeys.mealPlan("2026-03-01", "2026-03-31"),
  ];
  for (const key of recipes) {
    assert.equal(key[0], "recipes", `key ${JSON.stringify(key)} is not domain-prefixed`);
  }
});

test("an id is part of the key it identifies", () => {
  assert.notDeepEqual(queryKeys.recipe("a"), queryKeys.recipe("b"));
  assert.notDeepEqual(queryKeys.recipeComments("a"), queryKeys.recipeComments("b"));
});

test("different periods never share a key", () => {
  assert.notDeepEqual(
    queryKeys.mealPlan("2026-03-01", "2026-03-31"),
    queryKeys.mealPlan("2026-04-01", "2026-04-30"),
  );
});

/* ------------------------------------------------------------------------ finance */

test("every finance key starts with the finance domain segment", () => {
  const keys = [
    queryKeys.financeSettings(),
    queryKeys.financeOverview(august),
    queryKeys.financeMembers(),
    queryKeys.financeAccountsList(),
    queryKeys.financeAccount("a1"),
    queryKeys.financeCategoryTree(),
    queryKeys.financeTransactionsList(),
    queryKeys.financeTransactionFeed(),
    queryKeys.financeTransaction("t1"),
    queryKeys.financeTemplatesList(),
    queryKeys.financeBudgetsList(),
    queryKeys.financeHomeSummary(familyScope, august),
    queryKeys.financeGroupBreakdown("g1", familyScope, august),
    queryKeys.financeMemberBreakdown(august),
    queryKeys.financeSpendingSeries({ granularity: "month", bucketCount: 7 }),
    queryKeys.financeInsights(august),
    queryKeys.financeRecurringList(),
    queryKeys.financeRemindersList(),
    queryKeys.financeWidgetsList(),
    queryKeys.financeWidgetData(["w1"]),
  ];
  for (const key of keys) {
    assert.equal(key[0], "finance", `key ${JSON.stringify(key)} is not domain-prefixed`);
  }
  // The entity roots a mutation may invalidate wholesale are prefixed too.
  for (const root of [
    queryKeys.finance,
    queryKeys.financeAccounts,
    queryKeys.financeCategories,
    queryKeys.financeTransactions,
    queryKeys.financeTemplates,
    queryKeys.financeBudgets,
    queryKeys.financeAnalytics,
    queryKeys.financeRecurring,
    queryKeys.financeReminders,
    queryKeys.financeWidgets,
  ]) {
    assert.equal(root[0], "finance", `root ${JSON.stringify(root)} is not domain-prefixed`);
  }
});

test("the scope switcher separates cache entries", () => {
  const family = queryKeys.financeHomeSummary(familyScope, august);
  assert.notDeepEqual(family, queryKeys.financeHomeSummary(memberScope("m1"), august));
  assert.notDeepEqual(family, queryKeys.financeHomeSummary(accountScope("m1"), august));
  assert.notDeepEqual(
    queryKeys.financeHomeSummary(memberScope("m1"), august),
    queryKeys.financeHomeSummary(memberScope("m2"), august),
  );
});

test("two anchors in one month are one finance cache entry, two months are two", () => {
  assert.deepEqual(
    queryKeys.financeHomeSummary(familyScope, august),
    queryKeys.financeHomeSummary(familyScope, { granularity: "month", anchor: "2026-08-03" }),
  );
  assert.notDeepEqual(
    queryKeys.financeHomeSummary(familyScope, august),
    queryKeys.financeHomeSummary(familyScope, september),
  );
});

test("the expense/income tab is part of the key", () => {
  assert.notDeepEqual(
    queryKeys.financeHomeSummary(familyScope, august, 1),
    queryKeys.financeHomeSummary(familyScope, august, 2),
  );
});

test("equivalent transaction filters normalize to one key", () => {
  const a = normalizeTransactionFilters({
    memberIds: ["m2", "m1"],
    categoryIds: ["c1"],
    query: "  Coffee ",
  });
  const b = normalizeTransactionFilters({
    categoryIds: ["c1"],
    memberIds: ["m1", "m2"],
    query: "coffee",
  });
  assert.deepEqual(a, b);
  assert.deepEqual(queryKeys.financeTransactionsList({ memberIds: ["m2", "m1"] }), queryKeys
    .financeTransactionsList({ memberIds: ["m1", "m2"] }));
});

test("a filter that changes the result changes the key", () => {
  assert.notDeepEqual(
    queryKeys.financeTransactionsList({ accountIds: ["a1"] }),
    queryKeys.financeTransactionsList({ accountIds: ["a2"] }),
  );
  // The paged feed and the single-page list are different queries over the same filters.
  assert.notDeepEqual(
    queryKeys.financeTransactionsList({}),
    queryKeys.financeTransactionFeed({}),
  );
});

test("a widget refresh keys by the placement set, not the order it was rebuilt in", () => {
  assert.deepEqual(queryKeys.financeWidgetData(["w2", "w1"]), queryKeys.financeWidgetData(["w1", "w2"]));
  assert.notDeepEqual(queryKeys.financeWidgetData(["w1"]), queryKeys.financeWidgetData(["w1", "w2"]));
});
