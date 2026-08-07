import assert from "node:assert/strict";
import { test } from "node:test";

import {
  add,
  compare,
  convert,
  format,
  isZero,
  minorUnits,
  money,
  negate,
  parseAmount,
  subtract,
  sum,
  toInput,
  zero,
} from "./money.ts";

test("minorUnits knows the non-decimal currencies", () => {
  assert.equal(minorUnits("EUR"), 2);
  assert.equal(minorUnits("usd"), 2);
  assert.equal(minorUnits("JPY"), 0);
  assert.equal(minorUnits("KWD"), 3);
  assert.equal(minorUnits("ZZZ"), 2);
});

test("money truncates to whole minor units and upcases the code", () => {
  assert.deepEqual(money(10.7, "eur"), { amountMinor: 10, currencyCode: "EUR" });
});

test("arithmetic stays exact where floats would drift", () => {
  const a = money(10, "EUR");
  const b = money(20, "EUR");
  assert.equal(add(a, b).amountMinor, 30);
  assert.equal(subtract(b, a).amountMinor, 10);
  assert.equal(negate(a).amountMinor, -10);
  assert.ok(isZero(zero("EUR")));
});

test("mixing currencies throws instead of silently adding", () => {
  assert.throws(() => add(money(1, "EUR"), money(1, "USD")), /refusing to combine/);
});

test("sum folds a list", () => {
  const total = sum([money(100, "EUR"), money(250, "EUR"), money(-50, "EUR")], "EUR");
  assert.equal(total.amountMinor, 300);
  assert.equal(sum([], "EUR").amountMinor, 0);
});

test("compare orders amounts", () => {
  assert.equal(compare(money(1, "EUR"), money(2, "EUR")), -1);
  assert.equal(compare(money(2, "EUR"), money(1, "EUR")), 1);
  assert.equal(compare(money(2, "EUR"), money(2, "EUR")), 0);
});

test("parseAmount accepts the shapes users actually type", () => {
  assert.deepEqual(parseAmount("12", "EUR"), { amountMinor: 1200, currencyCode: "EUR" });
  assert.deepEqual(parseAmount("12.5", "EUR"), { amountMinor: 1250, currencyCode: "EUR" });
  assert.deepEqual(parseAmount("12,50", "EUR"), { amountMinor: 1250, currencyCode: "EUR" });
  assert.deepEqual(parseAmount(" 1234.56 ", "EUR"), { amountMinor: 123456, currencyCode: "EUR" });
  assert.deepEqual(parseAmount("-8.10", "EUR"), { amountMinor: -810, currencyCode: "EUR" });
  assert.deepEqual(parseAmount(".5", "EUR"), { amountMinor: 50, currencyCode: "EUR" });
  assert.deepEqual(parseAmount("100", "JPY"), { amountMinor: 100, currencyCode: "JPY" });
  assert.deepEqual(parseAmount("1.234", "KWD"), { amountMinor: 1234, currencyCode: "KWD" });
});

test("parseAmount truncates extra decimals rather than inventing money", () => {
  assert.deepEqual(parseAmount("1.999", "EUR"), { amountMinor: 199, currencyCode: "EUR" });
});

test("parseAmount rejects non-numbers", () => {
  for (const bad of ["", "  ", "abc", "1.2.3", "-", "1e5", "$5"]) {
    assert.equal(parseAmount(bad, "EUR"), null, `expected null for ${JSON.stringify(bad)}`);
  }
});

test("toInput round-trips parseAmount", () => {
  for (const input of ["0", "12.34", "-8.10", "1234.56"]) {
    const parsed = parseAmount(input, "EUR");
    assert.ok(parsed);
    assert.equal(toInput(parsed), Number(input).toFixed(2));
  }
  assert.equal(toInput(money(5, "JPY")), "5");
  assert.equal(toInput(money(5, "EUR")), "0.05");
});

test("convert rounds half up on the target minor unit", () => {
  // 10.00 EUR at 1.005 USD/EUR = 10.05 USD
  assert.deepEqual(convert(money(1000, "EUR"), "USD", 1.005), {
    amountMinor: 1005,
    currencyCode: "USD",
  });
  // Into a zero-decimal currency the result is whole units.
  assert.deepEqual(convert(money(1000, "EUR"), "JPY", 160.4), {
    amountMinor: 1604,
    currencyCode: "JPY",
  });
  assert.deepEqual(convert(money(-1000, "EUR"), "USD", 1.005), {
    amountMinor: -1005,
    currencyCode: "USD",
  });
});

test("format renders the currency and respects the minor unit", () => {
  assert.equal(format(money(123456, "EUR"), { locale: "en-US" }), "€1,234.56");
  assert.equal(format(money(1235, "JPY"), { locale: "en-US" }), "¥1,235");
  assert.equal(format(money(-810, "USD"), { locale: "en-US" }), "-$8.10");
  assert.equal(
    format(money(1000, "EUR"), { locale: "en-US", signDisplay: "always" }),
    "+€10.00",
  );
  assert.equal(format(money(123456, "EUR"), { locale: "en-US", hideSymbol: true }), "1,234.56");
});
