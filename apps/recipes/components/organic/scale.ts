export function scaleAmount(amount: string, factor: number): string {
  if (factor === 1) return amount;
  const match = /^\s*(\d+(?:[.,]\d+)?)\s*(.*)$/.exec(amount);
  if (!match) return amount;
  const value = parseFloat(match[1]!.replace(",", ".")) * factor;
  const rest = match[2] ?? "";
  return `${trim(value)}${rest === "" ? "" : ` ${rest}`}`;
}

function trim(value: number): string {
  const rounded = Math.round(value * 100) / 100;
  return Number.isInteger(rounded) ? `${rounded}` : `${rounded}`;
}
