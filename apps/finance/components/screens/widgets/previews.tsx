/**
 * The six widget previews. Each one is a static picture of what the placed widget shows —
 * the gallery adds nothing to the home screen itself (that is the launcher's own long-press
 * flow), so nothing here is a control.
 */
import type { ReactNode } from "react";
import { Text, View } from "react-native";

import {
  BudgetBar,
  DonutChart,
  Icon,
  IconCircle,
  MemberAvatar,
  MoneyText,
  formatCompact,
  iconOr,
  nocturne,
  seriesColor,
} from "@/components/nocturne";

import { WidgetAmount } from "./gallery.tsx";
import type {
  AccountsModel,
  BudgetsAndFamilyModel,
  CategoryModel,
  MonthModel,
  QuickAddModel,
  RecentLine,
} from "./data.ts";

/** Ten thousand: where the design stops printing an account balance in full. */
const COMPACT_FROM = 10_000;

function EmptyLine({ label }: { label: string }) {
  return <Text className="py-n2 text-[11px] text-neutral-600">{label}</Text>;
}

// ---- 4×2 · Швидке додавання -------------------------------------------------------------

export function QuickAddPreview({
  model,
  ownerLabel,
  otherLabel,
}: {
  model: QuickAddModel;
  /** "Шаблони · Сергій", already translated. */
  ownerLabel: string;
  /** "Інше" — the cell that opens the full add sheet. */
  otherLabel: string;
}) {
  return (
    <View>
      <View className="mb-n4 flex-row items-center justify-between">
        <Text className="text-[11px] font-medium text-neutral-400">{ownerLabel}</Text>
        <Icon name="arrow-clockwise" size={12} color={nocturne.neutral[600]} />
      </View>
      <View className="flex-row gap-n3">
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
        {/* The grid is four cells wide whatever the household has saved, so a member with one
            template does not get a full-width chip. */}
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
        borderColor: highlighted ? nocturne.accent[700] : nocturne.neutral[800],
        paddingVertical: nocturne.space.n3,
        paddingHorizontal: nocturne.space.n1,
      }}
    >
      <Icon
        name={iconOr(icon)}
        size={19}
        color={highlighted ? nocturne.accent[300] : nocturne.neutral[400]}
      />
      <Text
        className={`mt-[3px] text-[9.5px] font-medium ${highlighted ? "text-accent-300" : "text-neutral-400"}`}
        numberOfLines={1}
      >
        {label}
      </Text>
      {amount}
    </View>
  );
}

// ---- 2×2 · Місяць -----------------------------------------------------------------------

export function MonthPreview({
  model,
  overspentLabel,
}: {
  model: MonthModel;
  /** "2 бюджети", or null when nothing is over its limit. */
  overspentLabel: string | null;
}) {
  return (
    <View className="flex-row items-center gap-n4">
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
          <Text className="mt-[1px] text-[9.5px] text-overspend">{overspentLabel}</Text>
        ) : null}
      </View>
    </View>
  );
}

// ---- 2×1 · Категорія --------------------------------------------------------------------

export function CategoryPreview({
  model,
  emptyLabel,
}: {
  model: CategoryModel | null;
  emptyLabel: string;
}) {
  if (!model) return <EmptyLine label={emptyLabel} />;
  return (
    <View className="flex-row items-center gap-n3">
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

// ---- 4×3 · Бюджети + родина -------------------------------------------------------------

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
      <View className="gap-n3">
        {model.budgets.map((budget) => (
          <BudgetBar
            key={budget.id}
            spentMinor={budget.spent.amountMinor}
            limitMinor={budget.limit.amountMinor}
            label={budget.label}
            valueLabel={`${formatCompact(budget.spent)} / ${formatCompact(budget.limit, { hideSymbol: true })}`}
            height={5}
            color={nocturne.accent[400]}
          />
        ))}
      </View>
      {model.members.length > 0 ? (
        <View
          className="mt-n4 gap-n2 pt-n4"
          style={{ borderTopWidth: 1, borderTopColor: nocturne.neutral[800] }}
        >
          {model.members.map((member, index) => (
            <View key={member.id} className="flex-row items-center gap-n3">
              <MemberAvatar name={member.name} index={index} size={20} />
              <Text className="flex-1 text-[11px] text-neutral-400" numberOfLines={1}>
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

// ---- 4×2 · Останні операції -------------------------------------------------------------

export function RecentPreview({ lines, emptyLabel }: { lines: RecentLine[]; emptyLabel: string }) {
  if (lines.length === 0) return <EmptyLine label={emptyLabel} />;
  return (
    <View className="gap-n3">
      {lines.map((line, index) => (
        <View key={line.id} className="flex-row items-center gap-n3">
          <View
            style={{
              width: 8,
              height: 8,
              borderRadius: 4,
              backgroundColor: seriesColor(index),
            }}
          />
          <Text className="flex-1 text-[11.5px] text-neutral-300" numberOfLines={1}>
            {line.label}
          </Text>
          <MoneyText value={line.amount} size={11.5} />
        </View>
      ))}
    </View>
  );
}

// ---- 4×1 · Рахунки ----------------------------------------------------------------------

export function AccountsPreview({
  model,
  emptyLabel,
}: {
  model: AccountsModel;
  emptyLabel: string;
}) {
  if (model.accounts.length === 0) return <EmptyLine label={emptyLabel} />;
  return (
    <View className="flex-row gap-n5">
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
