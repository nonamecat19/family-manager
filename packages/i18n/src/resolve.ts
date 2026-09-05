/**
 * Locale resolution, kept free of `expo-localization` so it can be unit-tested under plain
 * node. `locale.ts` is the thin binding that feeds it the device's tag.
 */

/**
 * Resolution is prefix matching against the locales an app actually ships, never "whatever the
 * phone happens to be set to": a device set to German gets the app's default, not a half-
 * translated German. `locales` is the shipped set and `fallback` is the app's own default —
 * apps/finance defaults to Ukrainian and apps/recipes to English, which is a per-app choice
 * rather than a repo convention.
 *
 * Matching is longest-prefix-first, so a shipped `pt-BR` wins over a shipped `pt` for a
 * `pt-BR` device however the app happened to order its list.
 */
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
