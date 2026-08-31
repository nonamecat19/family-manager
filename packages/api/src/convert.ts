import { create } from "@bufbuild/protobuf";
import { type Money as WireMoney, MoneySchema } from "@fm/sdk/finance/v1/finance_pb";

import { money, type Money } from "./money.ts";

/**
 * The single wire↔domain boundary for money. `finance.v1.Money` is the only Money message in
 * the contracts, and every service that moves an amount imports it from there.
 *
 * protobuf-es maps int64 to bigint. Screens want plain numbers, so the conversion happens
 * exactly once — here — instead of every component deciding for itself. A JS number is an
 * exact integer below 2^53, which is ~90 trillion minor units; past that we have a different
 * problem than rounding.
 */
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
