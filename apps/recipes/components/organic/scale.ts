/**
 * Ingredient amounts are free text on the wire ("300", "1½", "a good pinch"), because a
 * family cookbook is not a database of grams. Scaling therefore multiplies the leading
 * number when there is one and otherwise leaves the text alone — "a good pinch" ×2 stays
 * "a good pinch", which is honest, where "NaN" would not be.
 */
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
