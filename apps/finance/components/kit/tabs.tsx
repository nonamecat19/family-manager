import type { Money } from "@fm/api";
import { Avatar, Icon, SegTabs } from "@fm/ui";
import { Pressable, ScrollView, Text, View } from "react-native";

import { useI18n } from "../i18n/index.tsx";
import { monthTitle } from "./format.ts";
import { organic } from "./tokens.ts";
import { MoneyText } from "./ui.tsx";

export interface SegmentedOption<T extends string> {
  value: T;
  label: string;
}

export interface SegmentedTabsProps<T extends string> {
  options: readonly SegmentedOption<T>[];
  value: T;
  onChange: (value: T) => void;
  className?: string;
}

/**
 * Finance passes `{ value, label }` pairs (labels come straight from i18n), so the shared
 * `SegTabs` — which takes `options` + a `labels` record — is adapted here rather than rewritten
 * at every call site. The chrome underneath is the shared one.
 */
export function SegmentedTabs<T extends string>({
  options,
  value,
  onChange,
  className = "",
}: SegmentedTabsProps<T>) {
  const labels = {} as Record<T, string>;
  for (const option of options) labels[option.value] = option.label;
  return (
    <View className={className}>
      <SegTabs options={options.map((option) => option.value)} value={value} onChange={onChange} labels={labels} />
    </View>
  );
}

export type PeriodTab = "day" | "week" | "month" | "year" | "custom";

export const PERIOD_TABS: readonly PeriodTab[] = ["day", "week", "month", "year", "custom"];

export interface PeriodTabsProps {
  value: PeriodTab;
  onChange: (value: PeriodTab) => void;
  options?: readonly PeriodTab[];
  className?: string;
}

export function PeriodTabs({ value, onChange, options = PERIOD_TABS, className = "" }: PeriodTabsProps) {
  const { t } = useI18n();
  const LABELS: Record<PeriodTab, string> = {
    day: t("common.period.day"),
    week: t("common.period.week"),
    month: t("common.period.month"),
    year: t("common.period.year"),
    custom: t("common.period.custom"),
  };
  return (
    <View className={`flex-row items-center justify-between ${className}`}>
      {options.map((option) => {
        const active = option === value;
        return (
          <Pressable
            key={option}
            accessibilityRole="tab"
            accessibilityLabel={LABELS[option]}
            accessibilityState={{ selected: active }}
            onPress={() => onChange(option)}
            className="items-center px-0.5 py-1.4"
          >
            <Text className={`text-12.5 ${active ? "font-fig-bold text-fg" : "font-fig text-neutral-600"}`}>
              {LABELS[option]}
            </Text>
            <View className={`mt-1.4 h-[2px] w-[18px] rounded-full ${active ? "bg-accent" : "bg-transparent"}`} />
          </Pressable>
        );
      })}
    </View>
  );
}

export interface PeriodStepperProps {
  label: string;
  onPrev: () => void;
  onNext: () => void;
  total?: Money;
  subtitle?: string;
  nextDisabled?: boolean;
  className?: string;
}

export function PeriodStepper({
  label,
  onPrev,
  onNext,
  total,
  subtitle,
  nextDisabled = false,
  className = "",
}: PeriodStepperProps) {
  return (
    <View className={`items-center ${className}`}>
      <View className="w-full flex-row items-center justify-between">
        <Pressable accessibilityRole="button" accessibilityLabel={label} onPress={onPrev} hitSlop={10}>
          <Icon name="caret-left" size={18} color={organic.neutral[600]} />
        </Pressable>
        <Text className="font-fig-med text-13 text-fg">{label}</Text>
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={label}
          onPress={onNext}
          disabled={nextDisabled}
          hitSlop={10}
          style={nextDisabled ? { opacity: 0.3 } : undefined}
        >
          <Icon name="caret-right" size={18} color={organic.neutral[600]} />
        </Pressable>
      </View>
      {total ? <MoneyText value={total} size={30} weight="medium" className="mt-2.1" /> : null}
      {subtitle ? <Text className="mt-0.5 font-fig text-11.5 text-neutral-600">{subtitle}</Text> : null}
    </View>
  );
}

export interface MonthStepperProps {
  year: number;
  month: number;
  onChange: (year: number, month: number) => void;
  total?: Money;
  subtitle?: string;
  nextDisabled?: boolean;
  className?: string;
}

export function MonthStepper({ year, month, onChange, ...rest }: MonthStepperProps) {
  const { t } = useI18n();
  const step = (delta: number) => {
    const zero = year * 12 + (month - 1) + delta;
    onChange(Math.floor(zero / 12), (zero % 12) + 1);
  };
  return (
    <PeriodStepper
      label={monthTitle(t, year, month)}
      onPrev={() => step(-1)}
      onNext={() => step(1)}
      {...rest}
    />
  );
}

export type Scope =
  | { kind: "family" }
  | { kind: "member"; id: string }
  | { kind: "account"; id: string };

export interface ScopeOption {
  id: string;
  name: string;
}

export interface ScopeSwitcherProps {
  scope: Scope;
  members: readonly ScopeOption[];
  accounts?: readonly ScopeOption[];
  onChange: (scope: Scope) => void;
  balance?: Money;
  onExpand?: () => void;
  className?: string;
}

export function scopeIsFamily(scope: Scope): boolean {
  return scope.kind === "family";
}

export function ScopeSwitcher({
  scope,
  members,
  accounts,
  onChange,
  balance,
  onExpand,
  className = "",
}: ScopeSwitcherProps) {
  const { t } = useI18n();

  const label =
    scope.kind === "family"
      ? t("home.scopeFamily")
      : scope.kind === "member"
        ? (members.find((m) => m.id === scope.id)?.name ?? t("home.scopeFamily"))
        : (accounts?.find((a) => a.id === scope.id)?.name ?? t("home.scopeFamily"));

  return (
    <View className={className}>
      <Pressable
        accessibilityRole={onExpand ? "button" : "header"}
        accessibilityLabel={label}
        onPress={onExpand}
        disabled={!onExpand}
        className="flex-row items-center gap-1.4"
      >
        <Text className="font-fig text-13 text-neutral-600">{label}</Text>
        {onExpand ? <Icon name="caret-down" size={14} color={organic.neutral[600]} /> : null}
      </Pressable>
      {balance ? <MoneyText value={balance} size={34} weight="medium" className="mt-1.4" /> : null}

      <ScrollView
        horizontal
        showsHorizontalScrollIndicator={false}
        contentContainerStyle={{ gap: 5.6 }}
        className="mt-2.8"
      >
        <ScopePill
          label={t("home.scopeAll", { count: members.length })}
          selected={scope.kind === "family"}
          onPress={() => onChange({ kind: "family" })}
        />
        {members.map((member, index) => (
          <ScopePill
            key={member.id}
            label={member.name}
            selected={scope.kind === "member" && scope.id === member.id}
            onPress={() => onChange({ kind: "member", id: member.id })}
            avatar={{ name: member.name, index }}
          />
        ))}
        {(accounts ?? []).map((account) => (
          <ScopePill
            key={account.id}
            label={account.name}
            selected={scope.kind === "account" && scope.id === account.id}
            onPress={() => onChange({ kind: "account", id: account.id })}
          />
        ))}
      </ScrollView>
    </View>
  );
}

function ScopePill({
  label,
  selected,
  onPress,
  avatar,
}: {
  label: string;
  selected: boolean;
  onPress: () => void;
  avatar?: { name: string; index: number };
}) {
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ selected }}
      onPress={onPress}
      className={`flex-none flex-row items-center gap-1.4 rounded-full py-1.25 pl-1.25 pr-2.8 ${
        selected ? "bg-accent" : "bg-surface shadow-card"
      }`}
    >
      {avatar ? (
        <Avatar name={avatar.name} index={avatar.index} size={22} />
      ) : (
        <View className="w-[3px]" />
      )}
      <Text className={`font-fig-med text-12 ${selected ? "text-white" : "text-neutral-600"}`}>
        {label}
      </Text>
    </Pressable>
  );
}
