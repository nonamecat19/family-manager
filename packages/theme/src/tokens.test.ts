import assert from "node:assert/strict";
import { createRequire } from "node:module";
import { test } from "node:test";

import { colors, spacing, radius } from "./tokens.ts";
import { categoryColor, darkScheme, lightScheme, schemeFor } from "./scheme.ts";

const require = createRequire(import.meta.url);
const preset = require("@fm/config/tailwind.preset.cjs") as {
  theme: {
    extend: {
      colors: Record<string, unknown>;
      spacing: Record<string, string>;
      borderRadius: Record<string, string>;
    };
  };
};

test("colour tokens match the tailwind preset", () => {
  const t = preset.theme.extend.colors;
  const primary = t.primary as { DEFAULT: string; fg: string; muted: string };

  assert.equal(primary.DEFAULT, colors.primary);
  assert.equal(primary.fg, colors.primaryFg);
  assert.equal(primary.muted, colors.primaryMuted);

  assert.equal(t.income, colors.income);
  assert.equal(t.expense, colors.expense);
  assert.equal(t.transfer, colors.transfer);

  assert.equal(t.bg, colors.bg);
  assert.equal(t.surface, colors.surface);
  assert.equal(t.border, colors.border);
  assert.equal(t.fg, colors.fg);
  assert.equal(t.muted, colors.muted);

  assert.equal(t["bg-dark"], colors.bgDark);
  assert.equal(t["surface-dark"], colors.surfaceDark);
  assert.equal(t["border-dark"], colors.borderDark);
  assert.equal(t["fg-dark"], colors.fgDark);
  assert.equal(t["muted-dark"], colors.mutedDark);
});

test("spacing and radius tokens match the tailwind preset", () => {
  for (const [key, value] of Object.entries(spacing)) {
    assert.equal(preset.theme.extend.spacing[key], `${value}px`, `spacing.${key}`);
  }
  for (const [key, value] of Object.entries(radius)) {
    const expected = key === "full" ? "9999px" : `${value}px`;
    assert.equal(preset.theme.extend.borderRadius[key], expected, `radius.${key}`);
  }
});

test("schemeFor picks dark only for dark", () => {
  assert.equal(schemeFor("dark"), darkScheme);
  assert.equal(schemeFor("light"), lightScheme);
  assert.equal(schemeFor(null), lightScheme);
  assert.equal(schemeFor(undefined), lightScheme);
});

test("income and expense keep their colours in both schemes", () => {
  assert.equal(lightScheme.income, darkScheme.income);
  assert.equal(lightScheme.expense, darkScheme.expense);
});

test("categoryColor prefers the category's own colour", () => {
  assert.equal(categoryColor("#123456", 0, ["#aaaaaa"]), "#123456");
  assert.equal(categoryColor("", 1, ["#aaaaaa", "#bbbbbb"]), "#bbbbbb");
  assert.equal(categoryColor(undefined, 3, ["#aaaaaa", "#bbbbbb"]), "#bbbbbb");
  assert.equal(categoryColor(undefined, 0, []), colors.muted);
});
