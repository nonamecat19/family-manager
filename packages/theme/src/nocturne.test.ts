import assert from "node:assert/strict";
import { createRequire } from "node:module";
import { test } from "node:test";

import { initialOf, nocturneCore } from "./nocturne.ts";

const require = createRequire(import.meta.url);
const preset = require("@fm/config/nocturne.preset.cjs") as {
  theme: {
    extend: {
      colors: Record<string, string | Record<string, string>>;
      borderRadius: Record<string, string>;
    };
  };
};

const colors = preset.theme.extend.colors;


test("the neutral ramp matches the tailwind preset", () => {
  assert.deepEqual(colors.neutral, { ...nocturneCore.neutral });
});

test("the accent ramp matches the tailwind preset", () => {
  assert.deepEqual(colors.accent, { ...nocturneCore.accent });
});

test("the ground roles match the tailwind preset", () => {
  assert.equal(colors.bg, nocturneCore.bg);
  assert.equal(colors.surface, nocturneCore.surface);
  assert.equal(colors.fg, nocturneCore.text);
  assert.equal(colors.divider, nocturneCore.divider);
  assert.equal(colors.border, nocturneCore.neutral[800]);
  assert.equal(colors.muted, nocturneCore.neutral[500]);
});

test("the radii match the tailwind preset", () => {
  assert.deepEqual(preset.theme.extend.borderRadius, {
    sm: `${nocturneCore.radius.sm}px`,
    md: `${nocturneCore.radius.md}px`,
    lg: `${nocturneCore.radius.lg}px`,
  });
});

test("Nocturne is dark-only: the *-dark roles resolve to the same values", () => {
  assert.equal(colors["bg-dark"], colors.bg);
  assert.equal(colors["surface-dark"], colors.surface);
  assert.equal(colors["border-dark"], colors.border);
  assert.equal(colors["fg-dark"], colors.fg);
  assert.equal(colors["muted-dark"], colors.muted);
});

test("the primary role is the accent, on the app ground", () => {
  assert.deepEqual(colors.primary, {
    DEFAULT: nocturneCore.accent.DEFAULT,
    fg: nocturneCore.bg,
    muted: nocturneCore.accent[900],
  });
});

test("error and expense both resolve to the one danger value", () => {
  assert.equal(colors.error, colors.overspend);
  assert.equal(colors.expense, colors.overspend);
});

test("initialOf takes the first character, uppercased", () => {
  assert.equal(initialOf("Олена"), "О");
  assert.equal(initialOf("sergiy"), "S");
  assert.equal(initialOf("  padded"), "P");
});

test("initialOf falls back rather than rendering an empty avatar", () => {
  assert.equal(initialOf(""), "?");
  assert.equal(initialOf("   "), "?");
});
