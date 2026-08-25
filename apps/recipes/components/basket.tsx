import type { BasketItem } from "@fm/api";
import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from "react";

/**
 * The cooking basket, lifted out of a single screen.
 *
 * The design splits what used to be one screen in two — the Plan tab holds the recipes and
 * how many batches of each, the List tab turns the same basket into a shopping list you tick
 * off. They are separate tabs, so the basket cannot live in either one's local state.
 *
 * Nothing here is persisted: `useSumIngredients` is a query over an ad-hoc basket, and the
 * server stores no basket of its own (see packages/api/src/recipes.ts).
 */
interface BasketContextValue {
  items: BasketItem[];
  /** Ingredient names already in the trolley, keyed by `name|unit`. */
  checked: Record<string, boolean>;
  add: (recipeId: string) => void;
  remove: (recipeId: string) => void;
  setBatches: (recipeId: string, batches: number) => void;
  batchesOf: (recipeId: string) => number;
  toggleChecked: (key: string) => void;
  clearChecked: () => void;
}

const BasketContext = createContext<BasketContextValue | null>(null);

export function BasketProvider({ children }: { children: ReactNode }) {
  const [items, setItems] = useState<BasketItem[]>([]);
  const [checked, setChecked] = useState<Record<string, boolean>>({});

  // Adding a recipe already in the basket is a no-op rather than a duplicate row — the
  // stepper is how you get more of it.
  const add = useCallback((recipeId: string) => {
    setItems((prev) =>
      prev.some((i) => i.recipeId === recipeId) ? prev : [...prev, { recipeId, servings: 0 }],
    );
  }, []);

  const remove = useCallback((recipeId: string) => {
    setItems((prev) => prev.filter((i) => i.recipeId !== recipeId));
  }, []);

  // servings 0 means "as written"; the stepper counts batches, so batch 1 == as written and
  // batch n == n × the recipe's own servings, resolved by the caller that knows the recipe.
  const setBatches = useCallback((recipeId: string, batches: number) => {
    setItems((prev) =>
      batches <= 0
        ? prev.filter((i) => i.recipeId !== recipeId)
        : prev.map((i) => (i.recipeId === recipeId ? { ...i, servings: batches === 1 ? 0 : batches } : i)),
    );
  }, []);

  const batchesOf = useCallback(
    (recipeId: string) => {
      const item = items.find((i) => i.recipeId === recipeId);
      if (!item) return 0;
      return item.servings === 0 ? 1 : item.servings;
    },
    [items],
  );

  const toggleChecked = useCallback((key: string) => {
    setChecked((prev) => ({ ...prev, [key]: !prev[key] }));
  }, []);

  const clearChecked = useCallback(() => setChecked({}), []);

  const value = useMemo(
    () => ({ items, checked, add, remove, setBatches, batchesOf, toggleChecked, clearChecked }),
    [items, checked, add, remove, setBatches, batchesOf, toggleChecked, clearChecked],
  );

  return <BasketContext.Provider value={value}>{children}</BasketContext.Provider>;
}

export function useBasket(): BasketContextValue {
  const ctx = useContext(BasketContext);
  if (!ctx) throw new Error("useBasket must be used inside a BasketProvider");
  return ctx;
}
