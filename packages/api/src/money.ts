const MINOR_UNITS: Record<string, number> = {
  BHD: 3,
  CLP: 0,
  IQD: 3,
  ISK: 0,
  JOD: 3,
  JPY: 0,
  KRW: 0,
  KWD: 3,
  LYD: 3,
  OMR: 3,
  PYG: 0,
  RWF: 0,
  TND: 3,
  UGX: 0,
  VND: 0,
  VUV: 0,
  XAF: 0,
  XOF: 0,
  XPF: 0,
};

export interface Money {
  amountMinor: number;
  currencyCode: string;
}

export function minorUnits(currencyCode: string): number {
  return MINOR_UNITS[currencyCode.toUpperCase()] ?? 2;
}

export function money(amountMinor: number, currencyCode: string): Money {
  return { amountMinor: Math.trunc(amountMinor), currencyCode: currencyCode.toUpperCase() };
}

export function zero(currencyCode: string): Money {
  return money(0, currencyCode);
}

export function isZero(m: Money): boolean {
  return m.amountMinor === 0;
}

export function negate(m: Money): Money {
  return money(-m.amountMinor, m.currencyCode);
}

export function abs(m: Money): Money {
  return money(Math.abs(m.amountMinor), m.currencyCode);
}

export function add(a: Money, b: Money): Money {
  assertSameCurrency(a, b);
  return money(a.amountMinor + b.amountMinor, a.currencyCode);
}

export function subtract(a: Money, b: Money): Money {
  assertSameCurrency(a, b);
  return money(a.amountMinor - b.amountMinor, a.currencyCode);
}

export function compare(a: Money, b: Money): number {
  assertSameCurrency(a, b);
  return a.amountMinor === b.amountMinor ? 0 : a.amountMinor < b.amountMinor ? -1 : 1;
}

export function sum(items: readonly Money[], currencyCode: string): Money {
  return items.reduce((acc, m) => add(acc, m), zero(currencyCode));
}

function assertSameCurrency(a: Money, b: Money): void {
  if (a.currencyCode !== b.currencyCode) {
    throw new Error(
      `money: refusing to combine ${a.currencyCode} and ${b.currencyCode} without an explicit rate`,
    );
  }
}

export function convert(m: Money, targetCurrency: string, rate: number): Money {
  const from = minorUnits(m.currencyCode);
  const to = minorUnits(targetCurrency);
  const scaled = (m.amountMinor / 10 ** from) * rate * 10 ** to;
  return money(roundHalfUp(scaled), targetCurrency);
}

function roundHalfUp(n: number): number {
  return n < 0 ? -Math.round(-n) : Math.round(n);
}

export function parseAmount(input: string, currencyCode: string): Money | null {
  const cleaned = input.trim().replace(/[\s\u00A0]/g, "").replace(",", ".");
  if (cleaned === "" || !/^-?\d*\.?\d*$/.test(cleaned) || !/\d/.test(cleaned)) return null;

  const negative = cleaned.startsWith("-");
  const [whole = "0", fraction = ""] = cleaned.replace("-", "").split(".");
  const decimals = minorUnits(currencyCode);

  const paddedFraction = fraction.padEnd(decimals, "0").slice(0, decimals);
  const minor = Number(whole) * 10 ** decimals + Number(paddedFraction || "0");
  if (!Number.isFinite(minor)) return null;

  return money(negative ? -minor : minor, currencyCode);
}

export function toInput(m: Money): string {
  const decimals = minorUnits(m.currencyCode);
  const negative = m.amountMinor < 0;
  const digits = Math.abs(m.amountMinor).toString().padStart(decimals + 1, "0");
  const whole = digits.slice(0, digits.length - decimals);
  const fraction = decimals > 0 ? `.${digits.slice(digits.length - decimals)}` : "";
  return `${negative ? "-" : ""}${whole}${fraction}`;
}

export interface FormatOptions {
  locale?: string;
  hideSymbol?: boolean;
  signDisplay?: "auto" | "always" | "never";
}

export function format(m: Money, opts: FormatOptions = {}): string {
  const decimals = minorUnits(m.currencyCode);
  const value = m.amountMinor / 10 ** decimals;

  const formatter = new Intl.NumberFormat(opts.locale, {
    style: opts.hideSymbol ? "decimal" : "currency",
    currency: m.currencyCode,
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals,
    signDisplay: opts.signDisplay ?? "auto",
  });
  return formatter.format(value);
}
