import { minorUnits, type Money } from "@fm/api";


const SYMBOLS: Record<string, string> = {
  UAH: "₴",
  USD: "$",
  EUR: "€",
  GBP: "£",
  PLN: "zł",
};

export function currencySymbol(code: string): string {
  return SYMBOLS[code.toUpperCase()] ?? `${code.toUpperCase()} `;
}

export const MINUS = "−";

export interface MoneyFormatOptions {
  sign?: "auto" | "always" | "never";
  hideSymbol?: boolean;
  decimals?: boolean;
}

function group(digits: string): string {
  return digits.replace(/\B(?=(\d{3})+(?!\d))/g, ",");
}

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

export function formatPercent(ratio: number): string {
  return `${Math.round(ratio * 100)}%`;
}

export function zeroLike(m: Money | undefined, currencyCode = "UAH"): Money {
  return { amountMinor: 0, currencyCode: m?.currencyCode ?? currencyCode };
}
