import assert from "node:assert/strict";
import { test } from "node:test";

import { normalizeBotUsername, telegramStartUrl } from "./telegram.ts";
import { queryKeys } from "./queryKeys.ts";

test("a bot username is accepted with or without its @", () => {
  assert.equal(normalizeBotUsername("fm_recipes_bot"), "fm_recipes_bot");
  assert.equal(normalizeBotUsername(" @FmRecipesBot "), "FmRecipesBot");
});

test("an unset or malformed bot username hides the feature", () => {
  for (const raw of [undefined, null, "", "@", "bot", "1recipes_bot", "fm-recipes-bot", "a".repeat(33)]) {
    assert.equal(normalizeBotUsername(raw), null, `accepted ${JSON.stringify(raw)}`);
  }
});

test("the start link carries the token as the start payload", () => {
  const token = "Zm9vYmFyZm9vYmFyZm9vYmFyZm9vYmFyZm9vYmFyMzI";
  assert.equal(
    telegramStartUrl("@fm_recipes_bot", token),
    `https://t.me/fm_recipes_bot?start=${token}`,
  );
});

test("a token Telegram would truncate or reject never becomes a link", () => {
  assert.throws(() => telegramStartUrl("fm_recipes_bot", "a+b/c="));
  assert.throws(() => telegramStartUrl("fm_recipes_bot", "a".repeat(65)));
  assert.throws(() => telegramStartUrl("fm_recipes_bot", ""));
  assert.throws(() => telegramStartUrl("not a bot", "abc"));
});

test("identities live under the auth domain", () => {
  assert.equal(queryKeys.identities()[0], "auth");
});
