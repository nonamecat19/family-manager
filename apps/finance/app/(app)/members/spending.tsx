/**
 * Screen 06 — Per-member spending ("Хто витрачає").
 *
 * Design: `docs/design/finance/Family Money Manager.dc.html`, block `id="s6"`. Three blocks
 * stacked on the app ground: the family split bar, the same split repeated per category group,
 * and the insight the server composed. One request feeds all three — `GetMemberBreakdown`
 * returns the total, the members, the group splits and the insights together, so the three
 * cards can never disagree about a month.
 *
 * Private accounts never reach this screen: `MemberSpending.private_account_count` is the only
 * fact about them that crosses the wire, and the design does not draw it, so nothing here can
 * leak one member's private spending into the family total.
 */
import {
  fromWire,
  InsightKind,
  monthPeriod,
  toDisplayError,
  useMemberBreakdown,
  type Insight,
  type Money,
} from "@fm/api";
import { useLocalSearchParams, useRouter } from "expo-router";
import { useMemo, useState } from "react";
import { Pressable, ScrollView, Text, View } from "react-native";

import { useI18n } from "@/components/i18n";
import {
  Button,
  Card,
  EmptyState,
  formatMoney,
  Icon,
  iconOr,
  Kicker,
  memberColor,
  MoneyText,
  monthName,
  monthNameLower,
  MonthStepper,
  nocturne,
  parseISO,
  Screen,
  ScreenHeader,
  Sheet,
  SplitBar,
  type SplitPart,
  type Translate,
} from "@/components/nocturne";

/** One member, already off the wire: the colour and the amount every block on this screen
 * shares. Both the split bar and the per-group bars index into this list, which is what keeps
 * Сергій the same purple in all six bars. */
interface MemberSlice {
  id: string;
  name: string;
  color: string;
  amount: Money;
}

function todayISO(): string {
  const now = new Date();
  const month = `${now.getMonth() + 1}`.padStart(2, "0");
  const day = `${now.getDate()}`.padStart(2, "0");
  return `${now.getFullYear()}-${month}-${day}`;
}

export default function PerMemberSpendingScreen() {
  const { t } = useI18n();
  const router = useRouter();
  // Screen 05 opens this screen on one member. The screen still shows the whole household —
  // the split is the point — but it says whose card was tapped and marks them in the legend,
  // so tapping Сергій and tapping Олена are not the same screen.
  const params = useLocalSearchParams<{ memberId?: string }>();
  const focusMemberId = typeof params.memberId === "string" ? params.memberId : "";
  const [anchor, setAnchor] = useState(todayISO);
  const [pickerOpen, setPickerOpen] = useState(false);

  const period = useMemo(() => monthPeriod(anchor), [anchor]);
  const breakdown = useMemberBreakdown(period);
  const { year, month } = parseISO(anchor);

  const data = breakdown.data;
  const currency = data?.total?.currencyCode || "UAH";
  const total = fromWire(data?.total, currency);

  const members: MemberSlice[] = useMemo(
    () =>
      (data?.members ?? []).map((entry, index) => ({
        id: entry.member?.userId ?? `${index}`,
        name: entry.member?.displayName ?? "",
        color: memberColor(entry.member?.avatarColorStep ?? index),
        amount: fromWire(entry.spent, currency),
      })),
    [data, currency],
  );

  const colorOf = useMemo(() => {
    const map = new Map(members.map((m) => [m.id, m.color]));
    return (memberId: string, fallbackIndex: number) => map.get(memberId) ?? memberColor(fallbackIndex);
  }, [members]);

  const focusMember = members.find((m) => m.id === focusMemberId);

  return (
    <Screen>
      <ScreenHeader
        gradient
        title={focusMember?.name ?? t("member.title")}
        subtitle={focusMember ? t("member.title") : undefined}
        leading={{ icon: "arrow-left", label: t("common.back"), onPress: () => router.back() }}
      >
        <MonthButton label={monthName(t, month)} onPress={() => setPickerOpen(true)} />
      </ScreenHeader>

      {breakdown.isPending ? (
        <Loading label={t("common.loadingEllipsis")} />
      ) : breakdown.isError ? (
        <LoadError error={breakdown.error} onRetry={() => void breakdown.refetch()} />
      ) : members.length === 0 || total.amountMinor === 0 ? (
        <View className="flex-1 justify-center">
          <EmptyState icon="chart-bar" title={t("charts.emptyTitle")} body={t("charts.emptyBody")} />
        </View>
      ) : (
        <ScrollView
          contentContainerStyle={{
            paddingHorizontal: nocturne.space.n4,
            paddingTop: nocturne.space.n5,
            paddingBottom: nocturne.space.n6,
            gap: nocturne.space.n4,
          }}
        >
          <SplitCard
            title={t("member.split")}
            total={total}
            members={members}
            highlightId={focusMember?.id ?? ""}
          />

          <View className="gap-n3">
            <Kicker className="ml-[2px]">{t("member.byGroup")}</Kicker>
            <Card className="gap-n5 border border-border">
              {(data?.groups ?? []).map((group) => (
                <View key={group.groupId}>
                  <View className="mb-n2 flex-row items-baseline justify-between">
                    <Text className="text-[12.5px] font-medium text-fg">{group.name}</Text>
                    <MoneyText value={fromWire(group.total, currency)} size={12.5} weight="regular" tone="muted" />
                  </View>
                  <SplitBar
                    height={8}
                    parts={group.members.map((share, index): SplitPart => ({
                      id: share.memberId || `${group.groupId}-${index}`,
                      label: members.find((m) => m.id === share.memberId)?.name ?? group.name,
                      value: fromWire(share.amount, currency).amountMinor,
                      color: colorOf(share.memberId, index),
                    }))}
                  />
                </View>
              ))}
            </Card>
          </View>

          {(data?.insights ?? []).length > 0 ? (
            (data?.insights ?? []).map((insight) => (
              <InsightCard key={insight.id} insight={insight} currency={currency} month={month} t={t} />
            ))
          ) : (
            <Card className="border border-border">
              <EmptyState icon="trend-up" title={t("member.emptyTitle")} body={t("member.emptyBody")} />
            </Card>
          )}
        </ScrollView>
      )}

      <Sheet visible={pickerOpen} onClose={() => setPickerOpen(false)} title={t("member.title")}>
        <MonthStepper
          year={year}
          month={month}
          total={breakdown.isSuccess ? total : undefined}
          onChange={(nextYear, nextMonth) =>
            setAnchor(`${nextYear}-${`${nextMonth}`.padStart(2, "0")}-01`)
          }
        />
      </Sheet>
    </Screen>
  );
}

/** The header's month affordance — "Серпень ⌄". `ScreenHeader` takes icon-only actions, so the
 * design's text-plus-caret control rides in the header's own children slot. */
function MonthButton({ label, onPress }: { label: string; onPress: () => void }) {
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      onPress={onPress}
      hitSlop={8}
      className="mt-n1 flex-row items-center justify-end gap-n2 self-end"
    >
      <Text className="text-[12px] text-neutral-400">{label}</Text>
      <Icon name="caret-down" size={10} color={nocturne.neutral[400]} />
    </Pressable>
  );
}

/** "Розподіл": the family total cut by member, then the same colours spelled out underneath. */
function SplitCard({
  title,
  total,
  members,
  highlightId,
}: {
  title: string;
  total: Money;
  members: readonly MemberSlice[];
  /** The member the screen was opened on, drawn brighter than the rest. */
  highlightId: string;
}) {
  return (
    <Card className="border border-border">
      <View className="mb-n4 flex-row items-baseline justify-between">
        <Text className="text-[13px] font-medium text-fg">{title}</Text>
        <MoneyText value={total} size={11} weight="regular" tone="muted" />
      </View>
      <SplitBar
        height={10}
        parts={members.map(
          (member): SplitPart => ({
            id: member.id,
            label: member.name,
            value: member.amount.amountMinor,
            color: member.color,
          }),
        )}
      />
      <View className="mt-n4 flex-row flex-wrap gap-n5">
        {members.map((member) => (
          <View key={member.id} className="min-w-[110px] flex-1 flex-row items-center gap-n3">
            <View
              className="h-[8px] w-[8px] rounded-full"
              style={{ backgroundColor: member.color }}
            />
            <View>
              <Text
                className={`text-[12.5px] font-medium ${
                  highlightId === "" || member.id === highlightId ? "text-fg" : "text-neutral-500"
                }`}
              >
                {member.name}
              </Text>
              <MoneyText value={member.amount} size={15} weight="regular" className="mt-[1px]" />
            </View>
          </View>
        ))}
      </View>
    </Card>
  );
}

/**
 * The insight callout. The sentence is composed server-side so app, widget and notification
 * say the same thing; when the server sends only the figures, the app spells them out itself
 * from `current`/`previous`/`ratio` rather than parsing a body.
 */
function InsightCard({
  insight,
  currency,
  month,
  t,
}: {
  insight: Insight;
  currency: string;
  month: number;
  t: Translate;
}) {
  const body = insight.body || composeBody(insight, currency, month, t);
  return (
    <Card className="flex-row items-start gap-n4 border border-border">
      <Icon name={iconOr(insight.icon, "trend-up")} size={18} color={nocturne.accent[400]} />
      <View className="flex-1">
        <Text className="mb-[2px] text-[12.5px] font-medium text-fg">{insight.title}</Text>
        {body ? <Text className="text-[11.5px] leading-[17px] text-neutral-500">{body}</Text> : null}
      </View>
    </Card>
  );
}

function composeBody(insight: Insight, currency: string, month: number, t: Translate): string {
  const sentences: string[] = [];
  if (insight.current && insight.previous) {
    const previousMonth = ((month - 2 + 12) % 12) + 1;
    sentences.push(
      t("member.insightComparison", {
        current: formatMoney(fromWire(insight.current, currency)),
        previous: formatMoney(fromWire(insight.previous, currency)),
        month: monthNameLower(t, previousMonth),
      }),
    );
  }
  if (insight.kind === InsightKind.BUDGET_EXCEEDED && insight.ratio > 0) {
    sentences.push(t("member.insightBudget", { percent: Math.round(insight.ratio * 100) }));
  }
  return sentences.join(" ");
}

function Loading({ label }: { label: string }) {
  return (
    <View className="flex-1 items-center justify-center">
      <Text className="text-[15px] text-neutral-500">{label}</Text>
    </View>
  );
}

function LoadError({ error, onRetry }: { error: unknown; onRetry: () => void }) {
  const { t } = useI18n();
  const shown = toDisplayError(error, t("common.loadFailed"));
  return (
    <View className="flex-1 justify-center gap-n4 px-n6">
      <Text className="text-[13.5px] leading-[21px] text-neutral-500">{shown.message}</Text>
      {shown.reference ? (
        <Text className="text-[12px] text-neutral-600">
          {t("common.errorReference", { ref: shown.reference })}
        </Text>
      ) : null}
      <Button title={t("common.tryAgain")} onPress={onRetry} />
    </View>
  );
}
