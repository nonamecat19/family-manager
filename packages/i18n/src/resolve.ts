export function localeFromTag<L extends string>(
  tag: string | null | undefined,
  locales: readonly L[],
  fallback: L,
): L {
  if (!tag) return fallback;
  const lower = tag.toLowerCase();
  let best: L | undefined;
  for (const locale of locales) {
    const candidate = locale.toLowerCase();
    if (!lower.startsWith(candidate)) continue;
    if (best === undefined || candidate.length > best.length) best = locale;
  }
  return best ?? fallback;
}
