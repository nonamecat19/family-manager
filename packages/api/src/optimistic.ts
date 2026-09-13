export interface UsageCounted {
  id: string;
  usageCount: number;
}

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

export function removeById<T extends { id: string }>(
  items: readonly T[] | undefined,
  id: string,
): readonly T[] | undefined {
  if (!items) return items;
  const next = items.filter((item) => item.id !== id);
  return next.length === items.length ? items : next;
}
