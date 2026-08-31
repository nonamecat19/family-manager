import type { Money } from "@fm/api";
import { Pressable, ScrollView, Text, View } from "react-native";

import { useI18n } from "../i18n/index.tsx";
import { monthTitle } from "./format.ts";
import { Icon } from "./icons.tsx";
import { nocturne } from "./tokens.ts";
import { MemberAvatar, MoneyText } from "./ui.tsx";

// ---- segmented control -------------------------------------------------------------------

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
 * The ВИТРАТИ / ДОХОДИ switch (and, on Charts, the three-way with ЗАГАЛЬНЕ in front). Generic
 * over the value so a screen keeps its own union — the kit does not know what an app "kind" is.
 *
 * Labels come from the caller, already translated: `t("common.expenses")` /
 * `t("common.income")` / `t("common.total")`. They are uppercase in the copy itself, not via
 * `text-transform`, because Ukrainian and English do not uppercase identically.
 */
export function SegmentedTabs<T extends string>({
  options,
  value,
  onChange,
  className = "",
}: SegmentedTabsProps<T>) {
  return (
    <View className={`flex-row rounded-md p-[3px] ${className}`} style={{ backgroundColor: "rgba(233,233,237,.06)" }}>
      {options.map((option) => {
        const active = option.value === value;
        return (
          <Pressable
            key={option.value}
            accessibilityRole="tab"
            accessibilityLabel={option.label}
            accessibilityState={{ selected: active }}
            onPress={() => onChange(option.value)}
            className={`flex-1 items-center rounded-sm py-n3 ${active ? "bg-accent-800" : ""}`}
          >
            <Text
              className={`text-[10.5px] font-semibold ${active ? "text-accent-200" : "text-neutral-500"}`}
              style={{ letterSpacing: 0.9 }}
            >
              {option.label}
            </Text>
          </Pressable>
        );
      })}
    </View>
  );
}

// ---- period tabs -------------------------------------------------------------------------

/** День / Тиждень / Місяць / Рік / Період. Maps 1:1 onto `@fm/api`'s `PeriodKind` minus
 * `"all"`, which these screens never offer. */
export type PeriodTab = "day" | "week" | "month" | "year" | "custom";

export const PERIOD_TABS: readonly PeriodTab[] = ["day", "week", "month", "year", "custom"];

export interface PeriodTabsProps {
  value: PeriodTab;
  onChange: (value: PeriodTab) => void;
  /** Narrow the strip — Home shows all five, a widget preview shows fewer. */
  options?: readonly PeriodTab[];
  className?: string;
}

/** The underlined period strip. Translates itself: the five labels are fixed copy, and a
 * screen passing them in could only get them wrong. */
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
            className="items-center px-[2px] py-n2"
          >
            <Text className={`text-[12.5px] ${active ? "font-medium text-fg" : "text-neutral-500"}`}>
              {LABELS[option]}
            </Text>
            <View
              className="mt-n2 h-[2px] w-[18px] rounded-full"
              style={{ backgroundColor: active ? nocturne.accent.DEFAULT : "transparent" }}
            />
          </Pressable>
        );
      })}
    </View>
  );
}

// ---- steppers ------------------------------------------------------------------------------

export interface PeriodStepperProps {
  /** Already-formatted label — "Серпень 2026", "29 серпня", "2026". */
  label: string;
  onPrev: () => void;
  onNext: () => void;
  /** The period's own figure, drawn under the label the way Home and Transactions do. */
  total?: Money;
  /** A line under the total — "5 груп". */
  subtitle?: string;
  /** Blocks stepping forward past the current period. */
  nextDisabled?: boolean;
  className?: string;
}

/** ‹ label › with the period's total under it. Generic over what the label says, so the same
 * control serves a day, a week, a month and a year. */
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
          <Icon name="caret-left" size={18} color={nocturne.neutral[500]} />
        </Pressable>
        <Text className="text-[13px] font-medium text-fg">{label}</Text>
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={label}
          onPress={onNext}
          disabled={nextDisabled}
          hitSlop={10}
          style={nextDisabled ? { opacity: 0.3 } : undefined}
        >
          <Icon name="caret-right" size={18} color={nocturne.neutral[500]} />
        </Pressable>
      </View>
      {total ? <MoneyText value={total} size={30} weight="medium" className="mt-n3" /> : null}
      {subtitle ? <Text className="mt-[2px] text-[11.5px] text-neutral-500">{subtitle}</Text> : null}
    </View>
  );
}

export interface MonthStepperProps {
  year: number;
  /** 1-12. */
  month: number;
  onChange: (year: number, month: number) => void;
  total?: Money;
  subtitle?: string;
  nextDisabled?: boolean;
  className?: string;
}

/** `PeriodStepper` bound to a month: it knows how to name one and how to roll a year over. */
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

// ---- scope switcher ---------------------------------------------------------------------------

/** What the figures on a screen are counting. `family` is everyone's shared view; the other
 * two narrow it to one member or one account. */
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
  /** Optional third rank of pills — the design offers accounts as a scope too. */
  accounts?: readonly ScopeOption[];
  onChange: (scope: Scope) => void;
  /** The balance shown beside the scope's name. */
  balance?: Money;
  /** Makes the title row a control (the caret next to "Родина"), for a fuller picker. */
  onExpand?: () => void;
  className?: string;
}

export function scopeIsFamily(scope: Scope): boolean {
  return scope.kind === "family";
}

/**
 * Родина | Сергій | Олена — the switch at the top of Home. Renders the current scope's name
 * and balance, then the pill row: an "Усі N" pill that resets to the family, one pill per
 * member, and (when given) one per account.
 */
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
        className="flex-row items-center gap-n2"
      >
        <Text className="text-[13px] text-neutral-400">{label}</Text>
        {onExpand ? <Icon name="caret-down" size={14} color={nocturne.neutral[500]} /> : null}
      </Pressable>
      {balance ? <MoneyText value={balance} size={34} weight="medium" className="mt-n2" /> : null}

      <ScrollView
        horizontal
        showsHorizontalScrollIndicator={false}
        contentContainerStyle={{ gap: nocturne.space.n2 }}
        className="mt-n4"
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
      className={`flex-none flex-row items-center gap-n2 rounded-full py-[5px] pl-[5px] pr-n4 ${
        selected ? "bg-accent-800" : "bg-surface"
      }`}
    >
      {avatar ? (
        <MemberAvatar name={avatar.name} index={avatar.index} size={22} />
      ) : (
        <View className="w-[3px]" />
      )}
      <Text className={`text-[12px] font-medium ${selected ? "text-accent-100" : "text-neutral-400"}`}>
        {label}
      </Text>
    </Pressable>
  );
}
