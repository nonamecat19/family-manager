import {
  parseAmount,
  toISODate,
  useAccounts,
  useCategories,
  type Money,
  type TransactionInput,
} from "@fm/api";
import type { Account, Category } from "@fm/sdk/finance/v1/finance_pb";
import { TransactionType } from "@fm/sdk/finance/v1/finance_pb";
import { Button, Card, Dot, Field, Loading } from "@fm/ui";
import { categoryColor, categoryPalette } from "@fm/theme";
import { useEffect, useMemo, useState } from "react";
import { Pressable, ScrollView, Text, View } from "react-native";

export interface TransactionFormProps {
  initial?: Partial<TransactionInput>;
  submitLabel: string;
  submitting: boolean;
  error?: string;
  onSubmit: (input: TransactionInput) => void;
  onDelete?: () => void;
}

const TYPES: readonly { label: string; value: TransactionType }[] = [
  { label: "Expense", value: TransactionType.EXPENSE },
  { label: "Income", value: TransactionType.INCOME },
  { label: "Transfer", value: TransactionType.TRANSFER },
];

/**
 * The add/edit form. The rules it enforces mirror the service exactly — a category matching
 * the type, a transfer needing a second account — so the user is told before a round trip,
 * not after one.
 */
export function TransactionForm({
  initial,
  submitLabel,
  submitting,
  error,
  onSubmit,
  onDelete,
}: TransactionFormProps) {
  const accounts = useAccounts();
  const [type, setType] = useState<TransactionType>(initial?.type ?? TransactionType.EXPENSE);
  const categories = useCategories(type === TransactionType.TRANSFER ? undefined : type);

  const [accountId, setAccountId] = useState(initial?.accountId ?? "");
  const [counterAccountId, setCounterAccountId] = useState(initial?.counterAccountId ?? "");
  const [categoryId, setCategoryId] = useState(initial?.categoryId ?? "");
  const [amountText, setAmountText] = useState(
    initial?.amount ? String(initial.amount.amountMinor / 100) : "",
  );
  const [note, setNote] = useState(initial?.note ?? "");
  const [day, setDay] = useState(initial?.occurredOn ?? toISODate(new Date()));

  const accountList = accounts.data?.accounts ?? [];
  const account = useMemo(
    () => accountList.find((a) => a.id === accountId),
    [accountList, accountId],
  );

  // Default to the first account rather than making the user pick when there is only one.
  useEffect(() => {
    if (accountId === "" && accountList.length > 0) setAccountId(accountList[0]!.id);
  }, [accountId, accountList]);

  // A category from the previous type would be rejected by the service; clear it on switch.
  useEffect(() => {
    if (type === TransactionType.TRANSFER) {
      setCategoryId("");
      return;
    }
    const list = categories.data?.categories ?? [];
    if (categoryId !== "" && !list.some((c) => c.id === categoryId)) setCategoryId("");
  }, [type, categories.data, categoryId]);

  const currency = account?.currencyCode ?? "EUR";
  const amount: Money | null = parseAmount(amountText, currency);

  const problems: string[] = [];
  if (amount === null || amount.amountMinor <= 0) problems.push("Enter an amount above zero.");
  if (accountId === "") problems.push("Pick an account.");
  if (type === TransactionType.TRANSFER) {
    if (counterAccountId === "") problems.push("Pick the destination account.");
    else if (counterAccountId === accountId) problems.push("A transfer needs two different accounts.");
    else {
      const other = accountList.find((a) => a.id === counterAccountId);
      if (other && other.currencyCode !== currency) {
        problems.push("Both accounts must use the same currency.");
      }
    }
  } else if (categoryId === "") {
    problems.push("Pick a category.");
  }

  if (accounts.isPending) return <Loading />;

  return (
    <ScrollView contentContainerClassName="gap-lg p-xl pb-2xl">
      <View className="flex-row gap-xs">
        {TYPES.map((t) => (
          <Pressable
            key={t.value}
            accessibilityRole="button"
            accessibilityState={{ selected: t.value === type }}
            onPress={() => setType(t.value)}
            className={`flex-1 rounded-md py-sm ${
              t.value === type ? "bg-primary" : "border border-border dark:border-border-dark"
            }`}
          >
            <Text
              className={`text-center text-body ${
                t.value === type ? "text-primary-fg" : "text-muted dark:text-muted-dark"
              }`}
            >
              {t.label}
            </Text>
          </Pressable>
        ))}
      </View>

      <Field
        label={`Amount (${currency})`}
        value={amountText}
        onChangeText={setAmountText}
        keyboardType="decimal-pad"
        placeholder="0.00"
      />

      <Picker
        label={type === TransactionType.TRANSFER ? "From account" : "Account"}
        items={accountList.map((a) => ({ id: a.id, label: a.name, color: "" }))}
        selected={accountId}
        onSelect={setAccountId}
      />

      {type === TransactionType.TRANSFER ? (
        <Picker
          label="To account"
          items={accountList
            .filter((a: Account) => a.id !== accountId)
            .map((a) => ({ id: a.id, label: a.name, color: "" }))}
          selected={counterAccountId}
          onSelect={setCounterAccountId}
        />
      ) : (
        <Picker
          label="Category"
          items={(categories.data?.categories ?? []).map((c: Category, i) => ({
            id: c.id,
            label: c.name,
            color: categoryColor(c.color, i, categoryPalette),
          }))}
          selected={categoryId}
          onSelect={setCategoryId}
          emptyHint="No categories yet — add one in Settings."
        />
      )}

      <Field label="Date" value={day} onChangeText={setDay} placeholder="YYYY-MM-DD" />
      <Field label="Note" value={note} onChangeText={setNote} placeholder="Optional" />

      {problems.length > 0 ? (
        <Text className="text-caption text-muted dark:text-muted-dark">{problems[0]}</Text>
      ) : null}
      {error ? <Text className="text-caption text-expense">{error}</Text> : null}

      <Button
        title={submitLabel}
        loading={submitting}
        disabled={problems.length > 0}
        onPress={() => {
          if (!amount) return;
          onSubmit({
            accountId,
            counterAccountId: type === TransactionType.TRANSFER ? counterAccountId : undefined,
            categoryId,
            type,
            amount,
            note: note.trim(),
            occurredOn: day,
          });
        }}
      />

      {onDelete ? <Button title="Delete" variant="danger" onPress={onDelete} /> : null}
    </ScrollView>
  );
}

function Picker({
  label,
  items,
  selected,
  onSelect,
  emptyHint,
}: {
  label: string;
  items: readonly { id: string; label: string; color: string }[];
  selected: string;
  onSelect: (id: string) => void;
  emptyHint?: string;
}) {
  return (
    <View className="gap-xs">
      <Text className="text-caption text-muted dark:text-muted-dark">{label}</Text>
      {items.length === 0 ? (
        <Text className="text-body text-muted dark:text-muted-dark">{emptyHint ?? "Nothing yet."}</Text>
      ) : (
        <View className="flex-row flex-wrap gap-xs">
          {items.map((item) => (
            <Pressable
              key={item.id}
              accessibilityRole="button"
              accessibilityState={{ selected: item.id === selected }}
              onPress={() => onSelect(item.id)}
            >
              <Card
                className={`flex-row items-center gap-xs px-md py-xs ${
                  item.id === selected ? "border-primary" : ""
                }`}
              >
                {item.color !== "" ? <Dot color={item.color} size={8} /> : null}
                <Text className="text-caption text-fg dark:text-fg-dark">{item.label}</Text>
              </Card>
            </Pressable>
          ))}
        </View>
      )}
    </View>
  );
}
