import { create } from "@bufbuild/protobuf";
import { type Money as WireMoney, MoneySchema } from "@fm/sdk/finance/v1/finance_pb";

import { money, type Money } from "./money.ts";

export type { WireMoney };

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
