import {
  DarkTheme,
  DefaultTheme,
  ThemeProvider as RNThemeProvider,
} from '@react-navigation/native';
import 'react-native-reanimated';
 
import { useTheme } from '@/hooks/useTheme';
import { Colors } from '@/theme/colors';
 
type Props = {
  children: React.ReactNode;
};
 
export const ThemeProvider = ({ children }: Props) => {
  const { effectiveTheme } = useTheme();
 
  // Create custom themes that use your Colors
  const customLightTheme = {
    ...DefaultTheme,
    colors: {
      ...DefaultTheme.colors,
      primary: Colors.light.primary,
      background: Colors.light.background,
      card: Colors.light.card,
      text: Colors.light.text,
      border: Colors.light.border,
      notification: Colors.light.red,
    },
  };
 
  const customDarkTheme = {
    ...DarkTheme,
    colors: {
      ...DarkTheme.colors,
      primary: Colors.dark.primary,
      background: Colors.dark.background,
      card: Colors.dark.card,
      text: Colors.dark.text,
      border: Colors.dark.border,
      notification: Colors.dark.red,
    },
  };

  const customBlueTheme = {
    ...DefaultTheme,
    colors: {
      ...DefaultTheme.colors,
      primary: Colors.blue.primary,
      background: Colors.blue.background,
      card: Colors.blue.card,
      text: Colors.blue.text,
      border: Colors.blue.border,
      notification: Colors.blue.red,
    },
  };
 
  const getTheme = () => {
    if (effectiveTheme === 'dark') return customDarkTheme;
    if (effectiveTheme === 'blue') return customBlueTheme;
    return customLightTheme;
  };
 
  return (
    <RNThemeProvider value={getTheme()}>
      {children}
    </RNThemeProvider>
  );
};

