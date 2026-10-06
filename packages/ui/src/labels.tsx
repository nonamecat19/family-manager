import { createContext, useContext, useMemo, type ReactNode } from "react";

export type UiTranslate = (key: string, vars?: Record<string, string | number>) => string;

const DEFAULT_LABELS: Record<string, string> = {
  "common.close": "Close",
  "ui.decrease": "Decrease {label}",
  "ui.increase": "Increase {label}",
  "ui.stars": "{count} stars",
  "ui.ratedOutOf5": "Rated {rating} out of 5",
  "nutrition.kcal": "kcal",
  "nutrition.protein": "protein",
  "nutrition.fat": "fat",
  "nutrition.carbs": "carbs",
  "nutrition.grams": "{value} g",
};

function fill(template: string, vars?: Record<string, string | number>): string {
  if (!vars) return template;
  return template.replace(/\{(\w+)\}/g, (whole, name: string) =>
    Object.hasOwn(vars, name) ? String(vars[name]) : whole,
  );
}

export const defaultUiTranslate: UiTranslate = (key, vars) => {
  const template = DEFAULT_LABELS[key];
  return template === undefined ? key : fill(template, vars);
};

const UiLabelsContext = createContext<UiTranslate>(defaultUiTranslate);

export interface UiLabelsProviderProps {
  translate: UiTranslate;
  children: ReactNode;
}

export function UiLabelsProvider({ translate, children }: UiLabelsProviderProps) {
  const value = useMemo(() => translate, [translate]);
  return <UiLabelsContext.Provider value={value}>{children}</UiLabelsContext.Provider>;
}

export function useUiTranslate(): UiTranslate {
  return useContext(UiLabelsContext);
}
