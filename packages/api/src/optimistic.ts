/**
 * Pure cache updaters used by the optimistic mutations in hooks.ts. They live outside the
 * hooks so they can be tested without a React tree or a QueryClient — the hook stays a thin
 * wiring layer, and the part that can actually be wrong is covered.
 */

/** The minimum a quick template must expose for an optimistic bump. */
export interface UsageCounted {
  id: string;
  usageCount: number;
}

/**
 * Tap-to-log has no confirmation step in the design: the chip must react on touch. This bumps
 * the tapped template's usage count in place and leaves the order alone — sort_order is the
 * server's, and re-sorting locally would make chips jump under the finger.
 *
 * Returns the same reference when nothing matched, so React Query does not repaint a list the
 * mutation did not touch.
 */
export function bumpTemplateUsage<T extends UsageCounted>(
  templates: readonly T[] | undefined,
  templateId: string,
): readonly T[] | undefined {
  if (!templates) return templates;
  let found = false;
  const next = templates.map((t) => {
    if (t.id !== templateId) return t;
    found = true;
    return { ...t, usageCount: t.usageCount + 1 };
  });
  return found ? next : templates;
}

/** Drops one item by id — the optimistic half of a delete. */
export function removeById<T extends { id: string }>(
  items: readonly T[] | undefined,
  id: string,
): readonly T[] | undefined {
  if (!items) return items;
  const next = items.filter((item) => item.id !== id);
  return next.length === items.length ? items : next;
}
