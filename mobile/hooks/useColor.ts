/**
 * Learn more about light and dark modes:
 * https://docs.expo.dev/guides/color-schemes/
 */
 
import { useTheme } from '@/hooks/useTheme';
import { Colors } from '@/theme/colors';
 
export function useColor(
  props: { light?: string; dark?: string; blue?: string },
  colorName: keyof typeof Colors.light & keyof typeof Colors.dark & keyof typeof Colors.blue
) {
  const { effectiveTheme, themeMode } = useTheme();
  const theme = themeMode === 'blue' ? 'blue' : effectiveTheme;
  const colorFromProps = props[theme as 'light' | 'dark' | 'blue'];
 
  if (colorFromProps) {
    return colorFromProps;
  } else {
    return Colors[theme as 'light' | 'dark' | 'blue'][colorName];
  }
}

