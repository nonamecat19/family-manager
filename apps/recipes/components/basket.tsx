import type { BasketItem } from "@fm/api";
import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from "react";

interface BasketContextValue {
  items: BasketItem[];
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

  const add = useCallback((recipeId: string) => {
    setItems((prev) =>
      prev.some((i) => i.recipeId === recipeId) ? prev : [...prev, { recipeId, servings: 0 }],
    );
  }, []);

  const remove = useCallback((recipeId: string) => {
    setItems((prev) => prev.filter((i) => i.recipeId !== recipeId));
  }, []);

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
