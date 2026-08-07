import { format, fromWire, type Money } from "@fm/api";
import type { Money as WireMoney } from "@fm/sdk/finance/v1/finance_pb";
import { TransactionType } from "@fm/sdk/finance/v1/finance_pb";
import { Text } from "react-native";

export interface AmountProps {
  value: Money | WireMoney | undefined;
  /** Colours the amount by direction. Omit for a neutral balance. */
  type?: TransactionType;
  size?: "body" | "title" | "amount";
  className?: string;
}

/**
 * The single place an amount becomes text. Colour follows direction — income green, expense
 * red, transfer blue — and a balance stays neutral so a negative balance is not "an expense".
 */
export function Amount({ value, type, size = "body", className = "" }: AmountProps) {
  const money = toMoney(value);
  const tone =
    type === TransactionType.INCOME
      ? "text-income"
      : type === TransactionType.EXPENSE
        ? "text-expense"
        : type === TransactionType.TRANSFER
          ? "text-transfer"
          : money.amountMinor < 0
            ? "text-expense"
            : "text-fg dark:text-fg-dark";

  const sign =
    type === TransactionType.EXPENSE ? "−" : type === TransactionType.INCOME ? "+" : "";

  return (
    <Text className={`text-${size} font-semibold ${tone} ${className}`}>
      {sign}
      {format(money)}
    </Text>
  );
}

function toMoney(value: Money | WireMoney | undefined): Money {
  if (!value) return { amountMinor: 0, currencyCode: "EUR" };
  // Wire messages carry bigint; plain Money carries number.
  if (typeof (value as WireMoney).amountMinor === "bigint") {
    return fromWire(value as WireMoney);
  }
  return value as Money;
}
