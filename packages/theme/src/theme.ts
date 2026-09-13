export interface Ramp {
  100: string;
  200: string;
  300: string;
  400: string;
  500: string;
  600: string;
  700: string;
  800: string;
  900: string;
}

export interface AccentRamp extends Ramp {
  DEFAULT: string;
}

export type SchemeName = "dark" | "light";

export interface Theme {
  name: string;
  scheme: SchemeName;

  bg: string;
  surface: string;
  text: string;
  muted: string;
  divider: string;

  neutral: Ramp;
  accent: AccentRamp;
  accent2: AccentRamp;

  danger: string;
  accentFg: string;
  dangerFg: string;

  radius: { sm: number; md: number; lg: number };

  fieldStyle?: "underline" | "outline" | "filled";

  labelCase?: "uppercase" | "sentence";

  fonts?: {
    body?: string;
    medium?: string;
    semibold?: string;
    display?: string;
  };
}
