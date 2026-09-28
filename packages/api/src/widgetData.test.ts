import assert from "node:assert/strict";
import { test } from "node:test";

import type { Clients } from "./client.ts";
import {
  financeWidgetSnapshots,
  mealPlanWidgetSnapshot,
  noteWidgetSnapshot,
} from "./widgetData.ts";

function moneyWire(amountMinor: number, currencyCode = "EUR") {
  return { amountMinor: BigInt(amountMinor), currencyCode };
}

function timestamp(ms: number) {
  return { seconds: BigInt(Math.floor(ms / 1000)), nanos: 0 };
}

function fakeClients(overrides: Record<string, unknown>): Clients {
  return overrides as unknown as Clients;
}

test("financeWidgetSnapshots zips each widget instance with its payload", async () => {
  const clients = fakeClients({
    finance: {
      listWidgets: async () => ({
        widgets: [
          { id: "w1", type: 5, size: 1 },
          { id: "w2", type: 2, size: 2 },
        ],
      }),
      getWidgetData: async (req: { widgetIds: string[] }) => {
        assert.deepEqual(req.widgetIds, ["w1", "w2"]);
        return {
          payloads: [
            {
              widgetId: "w1",
              type: 5,
              refreshedAt: timestamp(1_700_000_000_000),
              data: {
                case: "recentTransactions",
                value: {
                  transactions: Array.from({ length: 8 }, (_, i) => ({
                    id: `t${i}`,
                    type: 1,
                    merchant: "A".repeat(100),
                    note: "",
                    amount: moneyWire(-1200),
                    occurredOn: "2026-10-05",
                  })),
                },
              },
            },
            {
              widgetId: "w2",
              type: 2,
              refreshedAt: timestamp(1_700_000_100_000),
              data: {
                case: "month",
                value: {
                  label: "October",
                  periodTotal: moneyWire(50000),
                  budgetCount: 3,
                  budgetsWithinLimit: 2,
                },
              },
            },
          ],
        };
      },
    },
  });

  const snapshots = await financeWidgetSnapshots(clients);
  assert.equal(snapshots.length, 2);

  assert.equal(snapshots[0]?.widgetId, "w1");
  assert.equal(
    snapshots[0]?.refreshedAt,
    new Date(1_700_000_000_000).toISOString(),
  );
  assert.equal(snapshots[0]?.data.kind, "recentTransactions");
  if (snapshots[0]?.data.kind === "recentTransactions") {
    assert.equal(snapshots[0].data.transactions.length, 5);
    assert.ok(snapshots[0].data.transactions[0]!.merchant.length <= 60);
    assert.ok(snapshots[0].data.transactions[0]!.merchant.endsWith("…"));
  }

  assert.equal(snapshots[1]?.widgetId, "w2");
  assert.deepEqual(snapshots[1]?.data, {
    kind: "month",
    label: "October",
    periodTotal: { amountMinor: 50000, currencyCode: "EUR" },
    budgetCount: 3,
    budgetsWithinLimit: 2,
  });
});

test("financeWidgetSnapshots skips the data fetch when there are no widgets", async () => {
  let called = false;
  const clients = fakeClients({
    finance: {
      listWidgets: async () => ({ widgets: [] }),
      getWidgetData: async () => {
        called = true;
        return { payloads: [] };
      },
    },
  });

  const snapshots = await financeWidgetSnapshots(clients);
  assert.deepEqual(snapshots, []);
  assert.equal(called, false);
});

test("financeWidgetSnapshots gives a widget without a payload an empty snapshot", async () => {
  const clients = fakeClients({
    finance: {
      listWidgets: async () => ({ widgets: [{ id: "w1", type: 1, size: 1 }] }),
      getWidgetData: async () => ({ payloads: [] }),
    },
  });

  const snapshots = await financeWidgetSnapshots(clients);
  assert.equal(snapshots.length, 1);
  assert.equal(snapshots[0]?.refreshedAt, null);
  assert.deepEqual(snapshots[0]?.data, { kind: "empty" });
});

test("noteWidgetSnapshot fetches recent and starred notes, each bounded and truncated", async () => {
  const calls: Array<{ starredOnly: boolean }> = [];
  const longTitle = "T".repeat(100);

  const clients = fakeClients({
    notes: {
      listNotes: async (req: { starredOnly: boolean; pageSize: number }) => {
        calls.push({ starredOnly: req.starredOnly });
        assert.equal(req.pageSize, 5);
        const notes = Array.from({ length: 7 }, (_, i) => ({
          id: `n${i}`,
          title: longTitle,
          preview: "preview",
          starred: req.starredOnly,
          taskTotal: 2,
          taskDone: 1,
          updatedAt: timestamp(1_700_000_000_000),
        }));
        return { notes };
      },
    },
  });

  const snapshot = await noteWidgetSnapshot(clients);
  assert.deepEqual(calls, [{ starredOnly: false }, { starredOnly: true }]);
  assert.equal(snapshot.recent.length, 5);
  assert.equal(snapshot.starred.length, 5);
  assert.ok(snapshot.recent[0]!.title.length <= 60);
  assert.equal(snapshot.starred[0]!.starred, true);
});

test("mealPlanWidgetSnapshot joins meal plan entries with recipe titles for one day", async () => {
  const requestedDates: Array<{ fromDate: string; toDate: string }> = [];
  const requestedRecipeIds: string[] = [];

  const clients = fakeClients({
    recipes: {
      listMealPlan: async (req: { fromDate: string; toDate: string }) => {
        requestedDates.push(req);
        return {
          entries: [
            {
              id: "e1",
              recipeId: "r1",
              date: "2026-10-06",
              slot: 1,
              servings: 2,
            },
            {
              id: "e2",
              recipeId: "r2",
              date: "2026-10-06",
              slot: 2,
              servings: 4,
            },
          ],
        };
      },
      getRecipe: async (req: { recipeId: string }) => {
        requestedRecipeIds.push(req.recipeId);
        return {
          recipe: { id: req.recipeId, title: `Recipe ${req.recipeId}` },
        };
      },
    },
  });

  const snapshot = await mealPlanWidgetSnapshot(clients, "2026-10-06");
  assert.deepEqual(requestedDates, [
    { fromDate: "2026-10-06", toDate: "2026-10-06" },
  ]);
  assert.deepEqual(requestedRecipeIds.sort(), ["r1", "r2"]);
  assert.equal(snapshot.date, "2026-10-06");
  assert.deepEqual(snapshot.entries, [
    { entryId: "e1", recipeId: "r1", title: "Recipe r1", slot: 1, servings: 2 },
    { entryId: "e2", recipeId: "r2", title: "Recipe r2", slot: 2, servings: 4 },
  ]);
});

test("mealPlanWidgetSnapshot bounds the number of recipe lookups it makes", async () => {
  let lookups = 0;
  const clients = fakeClients({
    recipes: {
      listMealPlan: async () => ({
        entries: Array.from({ length: 9 }, (_, i) => ({
          id: `e${i}`,
          recipeId: `r${i}`,
          date: "2026-10-06",
          slot: 1,
          servings: 1,
        })),
      }),
      getRecipe: async (req: { recipeId: string }) => {
        lookups += 1;
        return { recipe: { id: req.recipeId, title: req.recipeId } };
      },
    },
  });

  const snapshot = await mealPlanWidgetSnapshot(clients, "2026-10-06");
  assert.equal(snapshot.entries.length, 5);
  assert.equal(lookups, 5);
});

function singlePayloadClients(data: { case: string; value: unknown }): Clients {
  return fakeClients({
    finance: {
      listWidgets: async () => ({ widgets: [{ id: "w1", type: 1, size: 1 }] }),
      getWidgetData: async () => ({
        payloads: [
          { widgetId: "w1", type: 1, refreshedAt: timestamp(0), data },
        ],
      }),
    },
  });
}

test("truncation never splits an emoji into a lone surrogate", async () => {
  const [snapshot] = await financeWidgetSnapshots(
    singlePayloadClients({
      case: "recentTransactions",
      value: {
        transactions: [
          {
            id: "t1",
            type: 1,
            merchant: `${"a".repeat(58)}😀😀😀`,
            note: "",
            amount: moneyWire(-1),
            occurredOn: "2026-10-06",
          },
        ],
      },
    }),
  );
  assert.ok(snapshot);
  assert.equal(snapshot.data.kind, "recentTransactions");
  if (snapshot.data.kind !== "recentTransactions") return;
  const merchant = snapshot.data.transactions[0]?.merchant ?? "";
  assert.doesNotMatch(merchant, /[\uD800-\uDBFF](?![\uDC00-\uDFFF])|(?<![\uD800-\uDBFF])[\uDC00-\uDFFF]/);
  assert.equal(merchant, `${"a".repeat(58)}😀…`);
});

test("accounts widget hides accounts excluded from the family total before bounding", async () => {
  const accounts = [
    ...Array.from({ length: 5 }, (_, i) => ({
      id: `x${i}`,
      name: `hidden ${i}`,
      kind: 1,
      balance: moneyWire(1),
      excludedFromFamilyTotal: true,
    })),
    {
      id: "a1",
      name: "Shared card",
      kind: 1,
      balance: moneyWire(500),
      excludedFromFamilyTotal: false,
    },
  ];
  const [snapshot] = await financeWidgetSnapshots(
    singlePayloadClients({
      case: "accounts",
      value: { accounts, sharedBalance: moneyWire(500) },
    }),
  );
  assert.ok(snapshot);
  if (snapshot.data.kind !== "accounts")
    return assert.fail("not an accounts snapshot");
  assert.deepEqual(
    snapshot.data.accounts.map((a) => a.id),
    ["a1"],
  );
});

test("mealPlanWidgetSnapshot keeps the other entries when one recipe lookup fails and orders by meal", async () => {
  const clients = fakeClients({
    recipes: {
      listMealPlan: async () => ({
        entries: [
          {
            id: "e3",
            recipeId: "r3",
            date: "2026-10-06",
            slot: 3,
            servings: 1,
          },
          {
            id: "e1",
            recipeId: "gone",
            date: "2026-10-06",
            slot: 1,
            servings: 1,
          },
          {
            id: "e2",
            recipeId: "r2",
            date: "2026-10-06",
            slot: 2,
            servings: 1,
          },
        ],
      }),
      getRecipe: async (req: { recipeId: string }) => {
        if (req.recipeId === "gone") throw new Error("not found");
        return {
          recipe: { id: req.recipeId, title: `Recipe ${req.recipeId}` },
        };
      },
    },
  });

  const snapshot = await mealPlanWidgetSnapshot(clients, "2026-10-06");
  assert.deepEqual(
    snapshot.entries.map((e) => [e.entryId, e.title]),
    [
      ["e1", ""],
      ["e2", "Recipe r2"],
      ["e3", "Recipe r3"],
    ],
  );
});
