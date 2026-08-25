import { organic } from "./tokens.ts";

/**
 * Aisle grouping for the shopping list.
 *
 * The service returns flat ingredient totals — there is no aisle on the wire, and inventing
 * one would mean a contract change and a taxonomy every family would disagree with. So the
 * grouping is a client-side reading of the ingredient name, with everything unmatched
 * falling into "Other" rather than being hidden or guessed at.
 */
export interface Aisle {
  name: string;
  dot: string;
  match: RegExp;
}

const AISLES: Aisle[] = [
  {
    name: "Produce",
    dot: organic.accent2[500],
    match:
      /mushroom|onion|carrot|garlic|potato|tomato|pepper|cabbage|beet|celery|dill|parsley|herb|salad|lettuce|cucumber|apple|lemon|lime|orange|berry|banana|ginger|leek|spinach|courgette|zucchini|aubergine|pumpkin|squash/i,
  },
  {
    name: "Dairy & eggs",
    dot: organic.accent[400],
    match: /milk|cream|butter|cheese|yog|yoghurt|yogurt|egg|kefir|curd|quark|ricotta|mascarpone/i,
  },
  {
    name: "Meat & fish",
    dot: organic.accent[600],
    match: /beef|pork|lamb|chicken|turkey|duck|bacon|sausage|ham|mince|fish|salmon|cod|tuna|shrimp|prawn|squid/i,
  },
  {
    name: "Bakery",
    dot: organic.accent[300],
    match: /bread|bun|roll|baguette|tortilla|pita|crouton|puff pastry|filo/i,
  },
  {
    name: "Pantry",
    dot: organic.neutral[500],
    match:
      /flour|sugar|salt|pepper|oil|vinegar|rice|barley|pasta|noodle|bean|lentil|chickpea|stock|broth|bay|spice|cumin|paprika|cinnamon|vanilla|yeast|soda|honey|soy|tomato paste|can|tin/i,
  },
];

const OTHER: Aisle = { name: "Other", dot: organic.neutral[400], match: /.^/ };

export function aisleFor(ingredientName: string): Aisle {
  return AISLES.find((a) => a.match.test(ingredientName)) ?? OTHER;
}

/** Groups totals into aisles, keeping aisle order stable and dropping empty ones. */
export function groupByAisle<T extends { name: string }>(items: T[]): { aisle: Aisle; items: T[] }[] {
  const buckets = new Map<string, { aisle: Aisle; items: T[] }>();
  for (const item of items) {
    const aisle = aisleFor(item.name);
    const bucket = buckets.get(aisle.name);
    if (bucket) bucket.items.push(item);
    else buckets.set(aisle.name, { aisle, items: [item] });
  }
  return [...AISLES, OTHER].map((a) => buckets.get(a.name)).filter((b): b is { aisle: Aisle; items: T[] } => b !== undefined);
}
