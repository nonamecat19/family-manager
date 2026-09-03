import assert from "node:assert/strict";
import { createRequire } from "node:module";
import { test } from "node:test";

import { nocturneCore } from "./nocturne.ts";
import { nocturneTheme, organicTheme, themes } from "./themes.ts";
import type { Theme } from "./theme.ts";

const require = createRequire(import.meta.url);

/**
 * Every theme must answer every role. A component drawn against the contract cannot check
 * whether the theme it was handed bothered to define `danger` — so these assert that neither
 * theme has a hole in it, for every theme at once rather than one by one.
 */
const RAMP_STEPS = [100, 200, 300, 400, 500, 600, 700, 800, 900] as const;
const COLOR_ROLES = ["bg", "surface", "text", "muted", "divider", "danger", "accentFg", "dangerFg"] as const;

for (const [name, theme] of Object.entries(themes) as [string, Theme][]) {
  test(`${name}: every colour role is a non-empty value`, () => {
    for (const role of COLOR_ROLES) {
      assert.equal(typeof theme[role], "string", role);
      assert.notEqual(theme[role], "", role);
    }
  });

  test(`${name}: the ramps are complete`, () => {
    for (const step of RAMP_STEPS) {
      assert.equal(typeof theme.neutral[step], "string", `neutral.${step}`);
      assert.equal(typeof theme.accent[step], "string", `accent.${step}`);
      assert.equal(typeof theme.accent2[step], "string", `accent2.${step}`);
    }
    assert.equal(typeof theme.accent.DEFAULT, "string");
    assert.equal(typeof theme.accent2.DEFAULT, "string");
  });

  test(`${name}: the three radius steps ascend`, () => {
    const { sm, md, lg } = theme.radius;
    assert.ok(sm > 0, "sm must be a real corner");
    assert.ok(md > sm, `md (${md}) must exceed sm (${sm})`);
    assert.ok(lg > md, `lg (${lg}) must exceed md (${md})`);
  });

  test(`${name}: its own name and scheme are declared`, () => {
    assert.equal(theme.name, name);
    assert.ok(theme.scheme === "dark" || theme.scheme === "light");
  });
}

test("the themes are actually different systems, not one palette twice", () => {
  // Guards against a copy-paste that would silently make every app look the same — the whole
  // point of the contract is that the difference survives it.
  assert.notEqual(nocturneTheme.bg, organicTheme.bg);
  assert.notEqual(nocturneTheme.accent.DEFAULT, organicTheme.accent.DEFAULT);
  assert.notEqual(nocturneTheme.scheme, organicTheme.scheme);
  assert.notEqual(nocturneTheme.radius.md, organicTheme.radius.md);
});

test("the nocturne theme is the shared nocturne core, not a second copy of it", () => {
  assert.equal(nocturneTheme.bg, nocturneCore.bg);
  assert.equal(nocturneTheme.surface, nocturneCore.surface);
  assert.equal(nocturneTheme.text, nocturneCore.text);
  assert.equal(nocturneTheme.muted, nocturneCore.neutral[500]);
  assert.deepEqual(nocturneTheme.accent, { ...nocturneCore.accent });
  assert.deepEqual(nocturneTheme.radius, { ...nocturneCore.radius });
});

/**
 * Organic's palette exists twice — here, and as CommonJS in the app's Tailwind config, which
 * cannot import ESM. Same trade as Nocturne's, same guard.
 */
test("the organic theme matches the recipes tailwind config", () => {
  const cfg = require("../../../apps/recipes/tailwind.config.js") as {
    theme: { extend: { colors: Record<string, string | Record<string, string>>; borderRadius: Record<string, string> } };
  };
  const colors = cfg.theme.extend.colors;

  assert.deepEqual(colors.neutral, { ...organicTheme.neutral });
  assert.deepEqual(colors.accent, { ...organicTheme.accent });
  assert.deepEqual(colors.accent2, { ...organicTheme.accent2 });
  assert.equal(colors.bg, organicTheme.bg);
  assert.equal(colors.surface, organicTheme.surface);
  assert.equal(colors.fg, organicTheme.text);
  assert.equal(colors.muted, organicTheme.muted);
  assert.equal(colors.divider, organicTheme.divider);
  assert.equal(colors.primary && (colors.primary as Record<string, string>).fg, organicTheme.accentFg);
  // Organic names its danger role `error`/`expense`; both point at the one warm red.
  assert.equal(colors.error, organicTheme.danger);
  assert.equal(colors.expense, organicTheme.danger);

  assert.deepEqual(cfg.theme.extend.borderRadius, {
    xl: `${organicTheme.radius.sm}px`,
    "2xl": `${organicTheme.radius.md}px`,
    "3xl": `${organicTheme.radius.lg}px`,
  });
});

/**
 * The treatments below are what stop "one component set" from meaning "one look". Each was
 * added because a shared component had silently restyled an app: the login fields rendered as
 * outlined boxes over finance's designed underlines, and Organic's sentence-case labels came
 * back uppercase. A default that is wrong for two of three apps is not a default.
 */
test("each theme declares its own input treatment", () => {
  // Nocturne's default is what apps/notes draws; apps/finance overrides it to "underline" in
  // its own root layout, which is why the base value is not asserted to be unique here.
  assert.equal(nocturneTheme.fieldStyle, "outline");
  assert.equal(organicTheme.fieldStyle, "filled");
});

test("label case is per theme, and the two systems disagree", () => {
  assert.equal(nocturneTheme.labelCase ?? "uppercase", "uppercase");
  assert.equal(organicTheme.labelCase, "sentence");
});

test("fonts are absent from the base themes, because two apps share one palette", () => {
  // apps/finance ships no font file and apps/notes loads Inter, yet both are Nocturne. Faces
  // therefore belong to the app's theme override, never to the shared system.
  assert.equal(nocturneTheme.fonts, undefined);
  assert.equal(organicTheme.fonts, undefined);
});
