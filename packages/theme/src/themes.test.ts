import assert from "node:assert/strict";
import { createRequire } from "node:module";
import { test } from "node:test";

import { organicTheme, themes } from "./themes.ts";
import type { Theme } from "./theme.ts";

const require = createRequire(import.meta.url);

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

test("the organic theme matches the organic tailwind preset", () => {
  const preset = require("../../../packages/config/organic.preset.cjs") as {
    theme: { extend: { colors: Record<string, string | Record<string, string>>; borderRadius: Record<string, string> } };
  };
  const colors = preset.theme.extend.colors;

  assert.deepEqual(colors.neutral, { ...organicTheme.neutral });
  assert.deepEqual(colors.accent, { ...organicTheme.accent });
  assert.deepEqual(colors.accent2, { ...organicTheme.accent2 });
  assert.equal(colors.bg, organicTheme.bg);
  assert.equal(colors.surface, organicTheme.surface);
  assert.equal(colors.fg, organicTheme.text);
  assert.equal(colors.muted, organicTheme.muted);
  assert.equal(colors.divider, organicTheme.divider);
  assert.equal(colors.primary && (colors.primary as Record<string, string>).fg, organicTheme.accentFg);
  assert.equal(colors.error, organicTheme.danger);
  assert.equal(colors.expense, organicTheme.danger);

  assert.deepEqual(preset.theme.extend.borderRadius, {
    xl: `${organicTheme.radius.sm}px`,
    "2xl": `${organicTheme.radius.md}px`,
    "3xl": `${organicTheme.radius.lg}px`,
  });
});

for (const app of ["finance", "notes", "recipes"]) test(`the ${app} app composes the organic preset instead of restating the palette`, () => {
  const cfg = require(`../../../apps/${app}/tailwind.config.js`) as {
    presets: { theme?: { extend?: { colors?: Record<string, string> } } }[];
    theme?: { extend?: { colors?: unknown } };
  };
  const composed = cfg.presets.some(
    (p) => p.theme?.extend?.colors?.bg === organicTheme.bg,
  );
  assert.ok(composed, `${app} must pull the organic palette in through a preset`);
  assert.equal(cfg.theme?.extend?.colors, undefined, "the palette must not be restated in the app config");
});

test("the organic theme declares its input treatment", () => {
  assert.equal(organicTheme.fieldStyle, "filled");
});

test("the organic theme uses sentence-case labels", () => {
  assert.equal(organicTheme.labelCase, "sentence");
});

test("fonts are absent from the base theme; apps add them through appTheme", () => {
  assert.equal(organicTheme.fonts, undefined);
});
