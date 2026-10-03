import type { ReactNode } from "react";
import { Text, View } from "react-native";

import { BudgetBar, DonutChart, formatCompact, IconCircle, MoneyText, seriesColor } from "@/components/kit";

import { WidgetAmount } from "./gallery.tsx";
import type {
  AccountsModel,
  BudgetsAndFamilyModel,
  CategoryModel,
  MonthModel,
  QuickAddModel,
  RecentLine,
} from "./data.ts";
import { Icon, Avatar, iconOr, organic } from "@fm/ui";

const COMPACT_FROM = 10_000;

function EmptyLine({ label }: { label: string }) {
  return <Text className="py-[5.6px] text-[11px] text-neutral-600">{label}</Text>;
}

export function QuickAddPreview({
  model,
  ownerLabel,
  otherLabel,
}: {
  model: QuickAddModel;
  ownerLabel: string;
  otherLabel: string;
}) {
  return (
    <View>
      <View className="mb-[11.2px] flex-row items-center justify-between">
        <Text className="text-[11px] font-fig-med text-neutral-600">{ownerLabel}</Text>
        <Icon name="arrow-clockwise" size={12} color={organic.neutral[600]} />
      </View>
      <View className="flex-row gap-[8.4px]">
        {model.templates.map((template, index) => (
          <TemplateCell
            key={template.id}
            icon={template.icon}
            label={template.label}
            amount={<MoneyText value={template.amount} size={9.5} weight="regular" tone="muted" />}
            highlighted={index === 0}
          />
        ))}
        <TemplateCell icon="plus" label={otherLabel} amount={<Dash />} highlighted={false} />
        {
}
        {Array.from({ length: Math.max(3 - model.templates.length, 0) }, (_, index) => (
          <View key={`spacer-${index}`} className="flex-1" />
        ))}
      </View>
    </View>
  );
}

function Dash() {
  return <Text className="text-[9.5px] text-neutral-600">—</Text>;
}

function TemplateCell({
  icon,
  label,
  amount,
  highlighted,
}: {
  icon: string;
  label: string;
  amount: ReactNode;
  highlighted: boolean;
}) {
  return (
    <View
      className="flex-1 items-center"
      style={{
        borderRadius: 10,
        borderWidth: 1,
        borderColor: highlighted ? organic.accent[700] : organic.neutral[800],
        paddingVertical: 8.4,
        paddingHorizontal: 2.8,
      }}
    >
      <Icon
        name={iconOr(icon)}
        size={19}
        color={highlighted ? organic.accent[300] : organic.neutral[600]}
      />
      <Text
        className={`mt-[3px] text-[9.5px] font-fig-med ${highlighted ? "text-accent-700" : "text-neutral-600"}`}
        numberOfLines={1}
      >
        {label}
      </Text>
      {amount}
    </View>
  );
}

export function MonthPreview({
  model,
  overspentLabel,
}: {
  model: MonthModel;
  overspentLabel: string | null;
}) {
  return (
    <View className="flex-row items-center gap-[11.2px]">
      <DonutChart
        segments={model.slices.map((slice) => ({
          id: slice.id,
          label: slice.id,
          value: slice.value,
        }))}
        size={52}
        thickness={11}
        gap={1}
      />
      <View className="flex-1">
        <Text className="text-[9.5px] text-neutral-600">{model.label}</Text>
        <MoneyText value={model.total} size={15} className="mt-[1px]" />
        {overspentLabel ? (
          <Text className="mt-[1px] text-[9.5px] text-error">{overspentLabel}</Text>
        ) : null}
      </View>
    </View>
  );
}

export function CategoryPreview({
  model,
  emptyLabel,
}: {
  model: CategoryModel | null;
  emptyLabel: string;
}) {
  if (!model) return <EmptyLine label={emptyLabel} />;
  return (
    <View className="flex-row items-center gap-[8.4px]">
      <IconCircle icon={model.icon} index={model.index} size={28} />
      <View className="flex-1">
        <Text className="text-[9.5px] text-neutral-600" numberOfLines={1}>
          {model.name}
        </Text>
        <MoneyText value={model.amount} size={13.5} className="mt-[1px]" />
      </View>
    </View>
  );
}

export function BudgetsAndFamilyPreview({
  model,
  emptyLabel,
}: {
  model: BudgetsAndFamilyModel;
  emptyLabel: string;
}) {
  const empty = model.budgets.length === 0 && model.members.length === 0;
  if (empty) return <EmptyLine label={emptyLabel} />;

  return (
    <View>
      <View className="gap-[8.4px]">
        {model.budgets.map((budget) => (
          <BudgetBar
            key={budget.id}
            spentMinor={budget.spent.amountMinor}
            limitMinor={budget.limit.amountMinor}
            label={budget.label}
            valueLabel={`${formatCompact(budget.spent)} / ${formatCompact(budget.limit, { hideSymbol: true })}`}
            height={5}
            color={organic.accent[600]}
          />
        ))}
      </View>
      {model.members.length > 0 ? (
        <View
          className="mt-[11.2px] gap-[5.6px] pt-[11.2px]"
          style={{ borderTopWidth: 1, borderTopColor: organic.neutral[800] }}
        >
          {model.members.map((member, index) => (
            <View key={member.id} className="flex-row items-center gap-[8.4px]">
              <Avatar name={member.name} index={index} size={20} />
              <Text className="flex-1 text-[11px] text-neutral-600" numberOfLines={1}>
                {member.name}
              </Text>
              <WidgetAmount value={member.spent} size={11} />
            </View>
          ))}
        </View>
      ) : null}
    </View>
  );
}

export function RecentPreview({ lines, emptyLabel }: { lines: RecentLine[]; emptyLabel: string }) {
  if (lines.length === 0) return <EmptyLine label={emptyLabel} />;
  return (
    <View className="gap-[8.4px]">
      {lines.map((line, index) => (
        <View key={line.id} className="flex-row items-center gap-[8.4px]">
          <View
            style={{
              width: 8,
              height: 8,
              borderRadius: 4,
              backgroundColor: seriesColor(index),
            }}
          />
          <Text className="flex-1 text-[11.5px] text-neutral-600" numberOfLines={1}>
            {line.label}
          </Text>
          <MoneyText value={line.amount} size={11.5} />
        </View>
      ))}
    </View>
  );
}

export function AccountsPreview({
  model,
  emptyLabel,
}: {
  model: AccountsModel;
  emptyLabel: string;
}) {
  if (model.accounts.length === 0) return <EmptyLine label={emptyLabel} />;
  return (
    <View className="flex-row gap-[16.8px]">
      {model.accounts.map((account) => (
        <View key={account.id} className="flex-1">
          <Text className="text-[9.5px] text-neutral-600" numberOfLines={1}>
            {account.name}
          </Text>
          <WidgetAmount
            value={account.balance}
            size={12.5}
            tone={account.balance.amountMinor < 0 ? "overspend" : "default"}
            compactFrom={COMPACT_FROM}
          />
        </View>
      ))}
    </View>
  );
}
