import { createContext, useContext, useMemo, type ReactNode } from "react";

import type { Theme, ThemeName } from "@fm/theme";
import { themes } from "@fm/theme";

/**
 * The theme a subtree is drawn in.
 *
 * This is what makes one component library serve apps that look nothing alike: a component
 * reads roles (`surface`, `accent`, `radius.md`) instead of colours, and the app decides which
 * system answers them. Switching the theme at runtime repaints everything below, which is why
 * the value is the theme object itself rather than a name components look up.
 */
const ThemeContext = createContext<Theme | null>(null);

export interface ThemeProviderProps {
  /** A theme object, or the name of one of the built-in systems. */
  theme: Theme | ThemeName;
  children: ReactNode;
}

export function ThemeProvider({ theme, children }: ThemeProviderProps) {
  const resolved = useMemo(() => (typeof theme === "string" ? themes[theme] : theme), [theme]);
  return <ThemeContext.Provider value={resolved}>{children}</ThemeContext.Provider>;
}

/**
 * The active theme. Throws when there is no provider above, rather than falling back to a
 * default: a silent fallback is how a screen ends up drawn in the wrong app's palette and
 * nobody notices until a screenshot. The message names the fix.
 */
export function useTheme(): Theme {
  const theme = useContext(ThemeContext);
  if (!theme) {
    throw new Error(
      "useTheme: no <ThemeProvider> above this component. Wrap the app's root layout in one, " +
        'e.g. <ThemeProvider theme="nocturne">.',
    );
  }
  return theme;
}
