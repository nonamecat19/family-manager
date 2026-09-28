export interface Tint {
  bg: string;
  fg: string;
}

export function initialOf(text: string): string {
  const first = text.trim()[0];
  return first ? first.toUpperCase() : "?";
}
