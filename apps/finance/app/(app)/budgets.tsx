import {
  abs,
  format,
  fromWire,
  parseAmount,
  toISODate,
  useBudgets,
  useCategories,
  useCreateBudget,
  useDeleteBudget,
  type Money,
} from "@fm/api";
import type { BudgetStatus, Money as WireMoney } from "@fm/sdk/finance/v1/finance_pb";
import { BudgetPeriod, TransactionType } from "@fm/sdk/finance/v1/finance_pb";
import { Button, Card, EmptyState, ErrorState, Field, Loading } from "@fm/ui";
import { useState } from "react";
import { Modal, Pressable, ScrollView, Text, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { Amount } from "../../components/Amount";

const PERIODS: readonly { label: string; value: BudgetPeriod }[] = [
  { label: "Weekly", value: BudgetPeriod.WEEK },
  { label: "Monthly", value: BudgetPeriod.MONTH },
  { label: "Yearly", value: BudgetPeriod.YEAR },
];

export default function BudgetsScreen() {
  const budgets = useBudgets();
  const [creating, setCreating] = useState(false);

  return (
    <SafeAreaView className="flex-1 bg-bg dark:bg-bg-dark">
      <ScrollView contentContainerClassName="gap-md p-lg pb-2xl">
        <Text className="text-title font-semibold text-fg dark:text-fg-dark">Budgets</Text>

        {budgets.isPending ? (
          <Loading />
        ) : budgets.isError ? (
          <ErrorState message={budgets.error.message} onRetry={() => void budgets.refetch()} />
        ) : budgets.data.budgets.length === 0 ? (
          <EmptyState
            title="No budgets yet"
            hint="Set a limit on a category, or on everything you spend."
            action={<Button title="Add budget" onPress={() => setCreating(true)} />}
          />
        ) : (
          <>
            {budgets.data.budgets.map((status) => (
              <BudgetRow key={status.budget?.id} status={status} />
            ))}
            <Button title="Add budget" variant="secondary" onPress={() => setCreating(true)} />
          </>
        )}
      </ScrollView>

      <NewBudgetModal visible={creating} onClose={() => setCreating(false)} />
    </SafeAreaView>
  );
}

function BudgetRow({ status }: { status: BudgetStatus }) {
  const remove = useDeleteBudget();
  const budget = status.budget;
  if (!budget) return null;

  // The bar is clamped at 100% so it stays inside its track, but the numbers above it are
  // not — an overspend is exactly what the user opened this screen to see.
  const filled = Math.min(status.share, 1);
  const tone = status.exceeded
    ? "bg-expense"
    : status.share > 0.8
      ? "bg-transfer"
      : "bg-primary";

  return (
    <Card className="gap-sm p-lg">
      <View className="flex-row items-center justify-between">
        <Text className="text-body text-fg dark:text-fg-dark">{budget.name}</Text>
        <Text className="text-caption text-muted dark:text-muted-dark">
          {periodLabel(budget.period)}
        </Text>
      </View>

      <View className="h-2 overflow-hidden rounded-full bg-border dark:bg-border-dark">
        <View className={`h-full rounded-full ${tone}`} style={{ width: `${filled * 100}%` }} />
      </View>

      <View className="flex-row items-baseline justify-between">
        <View className="flex-row items-baseline gap-xs">
          <Amount value={status.spent} />
          <Text className="text-caption text-muted dark:text-muted-dark">
            of {budget.limit ? format(fromWire(budget.limit)) : ""}
          </Text>
        </View>
        <Text
          className={`text-caption ${status.exceeded ? "text-expense" : "text-muted dark:text-muted-dark"}`}
        >
          {status.exceeded ? "over by " : ""}
          <Amount value={absMoney(status.remaining)} />
        </Text>
      </View>

      <View className="flex-row items-center justify-between">
        <Text className="text-caption text-muted dark:text-muted-dark">
          {status.period?.from} – {status.period?.to} · {status.daysRemaining}{" "}
          {status.daysRemaining === 1 ? "day" : "days"} left
        </Text>
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={`Delete ${budget.name}`}
          onPress={() => remove.mutate(budget.id)}
        >
          <Text className="text-caption text-expense">Delete</Text>
        </Pressable>
      </View>
    </Card>
  );
}

function NewBudgetModal({ visible, onClose }: { visible: boolean; onClose: () => void }) {
  const create = useCreateBudget();
  // Only expense categories can be budgeted; the service refuses the rest anyway.
  const categories = useCategories(TransactionType.EXPENSE);

  const [name, setName] = useState("");
  const [limitText, setLimitText] = useState("");
  const [period, setPeriod] = useState<BudgetPeriod>(BudgetPeriod.MONTH);
  const [categoryId, setCategoryId] = useState("");

  const limit = parseAmount(limitText, "EUR");
  const canSubmit = name.trim() !== "" && limit !== null && limit.amountMinor > 0;

  const submit = () => {
    if (!limit) return;
    create.mutate(
      {
        name: name.trim(),
        categoryId: categoryId === "" ? undefined : categoryId,
        limit,
        period,
        // Anchor on today: "500 a month starting now" is what someone setting a budget means.
        startOn: toISODate(new Date()),
      },
      {
        onSuccess: () => {
          setName("");
          setLimitText("");
          setCategoryId("");
          onClose();
        },
      },
    );
  };

  return (
    <Modal visible={visible} animationType="slide" onRequestClose={onClose}>
      <SafeAreaView className="flex-1 bg-bg dark:bg-bg-dark">
        <ScrollView contentContainerClassName="gap-lg p-xl">
          <Text className="text-title font-semibold text-fg dark:text-fg-dark">New budget</Text>

          <Field label="Name" value={name} onChangeText={setName} placeholder="Groceries" />
          <Field
            label="Limit"
            value={limitText}
            onChangeText={setLimitText}
            keyboardType="decimal-pad"
            placeholder="500.00"
            error={limitText !== "" && limit === null ? "Enter a number" : undefined}
          />

          <View className="gap-xs">
            <Text className="text-caption text-muted dark:text-muted-dark">Period</Text>
            <View className="flex-row gap-xs">
              {PERIODS.map((p) => (
                <Chip
                  key={p.value}
                  label={p.label}
                  active={p.value === period}
                  onPress={() => setPeriod(p.value)}
                />
              ))}
            </View>
          </View>

          <View className="gap-xs">
            <Text className="text-caption text-muted dark:text-muted-dark">Applies to</Text>
            <View className="flex-row flex-wrap gap-xs">
              <Chip
                label="Everything"
                active={categoryId === ""}
                onPress={() => setCategoryId("")}
              />
              {(categories.data?.categories ?? []).map((c) => (
                <Chip
                  key={c.id}
                  label={c.name}
                  active={c.id === categoryId}
                  onPress={() => setCategoryId(c.id)}
                />
              ))}
            </View>
          </View>

          {create.isError ? (
            <Text className="text-caption text-expense">{create.error.message}</Text>
          ) : null}

          <Button
            title="Create"
            loading={create.isPending}
            disabled={!canSubmit}
            onPress={submit}
          />
          <Button title="Cancel" variant="secondary" onPress={onClose} />
        </ScrollView>
      </SafeAreaView>
    </Modal>
  );
}

function Chip({
  label,
  active,
  onPress,
}: {
  label: string;
  active: boolean;
  onPress: () => void;
}) {
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityState={{ selected: active }}
      onPress={onPress}
      className={`rounded-md px-md py-xs ${
        active ? "bg-primary" : "border border-border dark:border-border-dark"
      }`}
    >
      <Text
        className={`text-caption ${active ? "text-primary-fg" : "text-muted dark:text-muted-dark"}`}
      >
        {label}
      </Text>
    </Pressable>
  );
}

function periodLabel(period: BudgetPeriod): string {
  return PERIODS.find((p) => p.value === period)?.label ?? "";
}

/** Remaining goes negative when overspent; the label already says "over by". */
function absMoney(m: WireMoney | undefined): Money | undefined {
  return m ? abs(fromWire(m)) : undefined;
}
