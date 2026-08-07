import { create } from "@bufbuild/protobuf";
import { MoneySchema, type Money as WireMoney } from "@fm/sdk/finance/v1/finance_pb";

import { money, type Money } from "./money.ts";

/**
 * protobuf-es maps int64 to bigint. Screens want plain numbers, so the boundary converts
 * exactly once — here — instead of every component deciding for itself.
 */
export function fromWire(m: WireMoney | undefined, fallbackCurrency = "EUR"): Money {
  if (!m) return money(0, fallbackCurrency);
  return money(Number(m.amountMinor), m.currencyCode || fallbackCurrency);
}

export function toWire(m: Money): WireMoney {
  return create(MoneySchema, {
    amountMinor: BigInt(Math.trunc(m.amountMinor)),
    currencyCode: m.currencyCode,
  });
}
