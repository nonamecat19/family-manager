import { minorUnits, type Money } from "@fm/api";

/**
 * Money as the design draws it: `₴17,634`, `−₴10,835`, `$1,420`, `₴15.7K`.
 *
 * Deliberately NOT `Intl.NumberFormat` (which `@fm/api`'s own `format()` uses). Hermes ships
 * a cut-down ICU on Android and the currency formats it produces are locale-dependent — a
 * Ukrainian device renders `17 634,00 ₴`, trailing symbol, non-breaking space, always two
 * decimals. The design puts the symbol in front, groups with commas and shows decimals only
 * when the amount actually has them, on every screen and in both languages. That is a fixed
 * visual grammar, not a locale preference, so it is written out here.
 *
 * `@fm/api`'s `format()` is still the right call anywhere the OS locale should win (an export,
 * a share sheet); nothing on these eleven screens is that.
 */

const SYMBOLS: Record<string, string> = {
  UAH: "₴",
  USD: "$",
  EUR: "€",
  GBP: "£",
  PLN: "zł",
};

/** Falls back to the ISO code plus a space — an unknown currency reads as `CHF 1,420`, which
 * is wrong-looking but never silently wrong. */
export function currencySymbol(code: string): string {
  return SYMBOLS[code.toUpperCase()] ?? `${code.toUpperCase()} `;
}

/** U+2212 MINUS SIGN, not a hyphen: at these weights a hyphen reads as a dash in a list. */
export const MINUS = "−";

export interface MoneyFormatOptions {
  /** `auto` (default) prints a minus for negatives; `always` also prints `+`; `never` prints
   * the magnitude, for a row that carries direction some other way. */
  sign?: "auto" | "always" | "never";
  /** Drop the currency symbol — for a column that already names the currency once. */
  hideSymbol?: boolean;
  /** Force fraction digits on (`₴50.00`) or off (`₴50`). Default: shown only when the amount
   * is not a whole unit, which is what the design does. */
  decimals?: boolean;
}

function group(digits: string): string {
  return digits.replace(/\B(?=(\d{3})+(?!\d))/g, ",");
}

/** The workhorse. `amountMinor` is integer minor units — never a float, never a bigint. */
export function formatMinor(
  amountMinor: number,
  currencyCode: string,
  opts: MoneyFormatOptions = {},
): string {
  const places = minorUnits(currencyCode);
  const scale = 10 ** places;
  const negative = amountMinor < 0;
  const magnitude = Math.abs(Math.trunc(amountMinor));
  const whole = Math.trunc(magnitude / scale);
  const fraction = magnitude % scale;

  const showFraction = opts.decimals ?? (places > 0 && fraction !== 0);
  const body =
    group(String(whole)) + (showFraction ? `.${String(fraction).padStart(places, "0")}` : "");

  const sign = opts.sign === "never" ? "" : negative ? MINUS : opts.sign === "always" ? "+" : "";
  const symbol = opts.hideSymbol ? "" : currencySymbol(currencyCode);
  return `${sign}${symbol}${body}`;
}

export function formatMoney(m: Money, opts: MoneyFormatOptions = {}): string {
  return formatMinor(m.amountMinor, m.currencyCode, opts);
}

/**
 * The widget scale — `₴15.7K`, `₴1.2M`. Widgets are 2-4 columns wide and the design shortens
 * there and only there; a full screen always prints the whole number.
 */
export function formatCompact(m: Money, opts: MoneyFormatOptions = {}): string {
  const places = minorUnits(m.currencyCode);
  const units = Math.abs(m.amountMinor) / 10 ** places;
  const sign = opts.sign === "never" ? "" : m.amountMinor < 0 ? MINUS : opts.sign === "always" ? "+" : "";
  const symbol = opts.hideSymbol ? "" : currencySymbol(m.currencyCode);

  const step = (value: number, suffix: string): string => {
    const shown = value < 10 ? Math.round(value * 10) / 10 : Math.round(value);
    return `${sign}${symbol}${shown}${suffix}`;
  };
  if (units >= 1_000_000) return step(units / 1_000_000, "M");
  if (units >= 1_000) return step(units / 1_000, "K");
  return formatMoney(m, opts);
}

/** `65%`. Ratios arrive unclamped (a 256% budget is a real state) and are rounded, not floored —
 * a budget at 99.6% must not read as 99%. */
export function formatPercent(ratio: number): string {
  return `${Math.round(ratio * 100)}%`;
}

/** A zero of the same currency — what a screen shows before its first query resolves, so the
 * layout does not jump when the real figure lands. */
export function zeroLike(m: Money | undefined, currencyCode = "UAH"): Money {
  return { amountMinor: 0, currencyCode: m?.currencyCode ?? currencyCode };
}
