import { createContext, useContext, useMemo, type ReactNode } from "react";

import type { Theme, ThemeName } from "@fm/theme";
import { themes } from "@fm/theme";

const ThemeContext = createContext<Theme | null>(null);

export interface ThemeProviderProps {
  theme: Theme | ThemeName;
  children: ReactNode;
}

export function ThemeProvider({ theme, children }: ThemeProviderProps) {
  const resolved = useMemo(() => (typeof theme === "string" ? themes[theme] : theme), [theme]);
  return <ThemeContext.Provider value={resolved}>{children}</ThemeContext.Provider>;
}

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
