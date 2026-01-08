import { useState, useEffect } from 'react';
import AsyncStorage from '@react-native-async-storage/async-storage';
import { useColorScheme as useRNColorScheme } from 'react-native';

export type ThemeMode = 'light' | 'dark' | 'auto' | 'blue';

const THEME_STORAGE_KEY = 'app_theme_preference';

export function useTheme() {
  const systemColorScheme = useRNColorScheme();
  const [themeMode, setThemeMode] = useState<ThemeMode>('auto');
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    loadTheme();
  }, []);

  async function loadTheme() {
    try {
      const stored = await AsyncStorage.getItem(THEME_STORAGE_KEY);
      if (stored && (stored === 'light' || stored === 'dark' || stored === 'auto' || stored === 'blue')) {
        setThemeMode(stored as ThemeMode);
      }
    } catch (error) {
      console.error('Failed to load theme preference:', error);
    } finally {
      setIsLoading(false);
    }
  }

  async function setTheme(newTheme: ThemeMode) {
    try {
      await AsyncStorage.setItem(THEME_STORAGE_KEY, newTheme);
      setThemeMode(newTheme);
    } catch (error) {
      console.error('Failed to save theme preference:', error);
    }
  }

  // Determine the actual theme to use
  const effectiveTheme = themeMode === 'auto' 
    ? (systemColorScheme || 'light')
    : themeMode === 'blue'
    ? 'light' // Blue theme uses light base
    : themeMode;

  return {
    themeMode,
    effectiveTheme,
    setTheme,
    isLoading,
  };
}

