import {
  currentPeriod,
  fromWire,
  MemberRole,
  MemberStatus,
  toDisplayError,
  useFamily,
  useFinanceSettings,
  useHouseholdOverview,
  useInviteMember,
  useSetOverspendNotifications,
  type Money,
} from "@fm/api";
import { useRouter } from "expo-router";
import { useMemo, useState } from "react";
import { ScrollView, Text, View } from "react-native";

import { useI18n } from "@/components/i18n";
import {
  Badge,
  BottomIndicator,
  Button,
  Card,
  Divider,
  Drawer,
  EmptyState,
  Field,
  Icon,
  Kicker,
  ListSection,
  MemberAvatar,
  MoneyText,
  monthNameLower,
  nocturne,
  parseISO,
  Row,
  Screen,
  ScreenHeader,
  Sheet,
  Stat,
  ToggleRow,
  useDrawerItems,
  formatPercent,
} from "@/components/nocturne";

/**
 * Screen 05 — Родина. The household in one card stack: the two headline figures, one card per
 * member (spend, share, transaction count), the invite affordance, and the three shared
 * settings.
 *
 * Everything on it comes from a single `GetHouseholdOverview` round trip, which is also why
 * private accounts cannot leak into the family totals here: the server sends a shared balance
 * and, per member, nothing about their private accounts but a count.
 */
export default function HouseholdScreen() {
  const { t } = useI18n();
  const router = useRouter();

  // The design's header reads "Витрати · серпень": the current calendar month, not a period
  // the user can step on this screen.
  const period = useMemo(() => currentPeriod("month"), []);
  const anchorMonth = "anchor" in period ? parseISO(period.anchor).month : new Date().getMonth() + 1;

  const overview = useHouseholdOverview(period);
  const family = useFamily();
  const settings = useFinanceSettings();
  const invite = useInviteMember();
  const setOverspend = useSetOverspendNotifications();

  const [menuOpen, setMenuOpen] = useState(false);
  const [inviteOpen, setInviteOpen] = useState(false);
  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteError, setInviteError] = useState<string | null>(null);
  // The switch has to answer the tap before the round trip does; the query is still the truth
  // once it lands.
  const [overspendPending, setOverspendPending] = useState<boolean | null>(null);

  const drawerItems = useDrawerItems();
  const currency = settings.data?.baseCurrencyCode ?? "UAH";
  const data = overview.data;
  const familyName = family.data?.family?.name ?? t("household.title");
  const members = data?.members ?? [];

  const sharedBalance = fromWire(data?.sharedBalance, currency);
  const periodExpense = fromWire(data?.periodExpense, currency);
  const overspendOn = overspendPending ?? data?.overspendNotificationsEnabled ?? false;

  const openInvite = () => {
    setInviteEmail("");
    setInviteError(null);
    setInviteOpen(true);
  };

  const submitInvite = async () => {
    const familyId = family.data?.family?.id ?? "";
    if (familyId === "" || inviteEmail.trim() === "") return;
    setInviteError(null);
    try {
      await invite.mutateAsync({ familyId, email: inviteEmail.trim() });
      setInviteOpen(false);
    } catch (error) {
      setInviteError(toDisplayError(error, t("household.inviteError")).message);
    }
  };

  const toggleOverspend = (next: boolean) => {
    setOverspendPending(next);
    setOverspend.mutate(next, {
      onSettled: () => setOverspendPending(null),
    });
  };

  return (
    <Screen>
      <ScreenHeader
        gradient
        title={familyName}
        leading={{ icon: "list", label: t("nav.menu"), onPress: () => setMenuOpen(true) }}
        trailing={{ icon: "gear", label: t("nav.settings"), onPress: () => router.push("/(app)/settings") }}
      >
        <View className="mt-n4 flex-row gap-n5">
          <View>
            <Kicker>{t("household.sharedBalance")}</Kicker>
            <MoneyText value={sharedBalance} size={22} className="mt-[2px]" />
          </View>
          <View className="w-[1px]" style={{ backgroundColor: nocturne.divider }} />
          <View>
            <Kicker>{t("household.spendingIn", { month: monthNameLower(t, anchorMonth) })}</Kicker>
            <MoneyText value={periodExpense} size={22} className="mt-[2px]" />
          </View>
        </View>
      </ScreenHeader>

      {overview.isPending ? (
        <View className="flex-1 items-center justify-center">
          <Text className="text-[13px] text-neutral-500">{t("common.loadingEllipsis")}</Text>
        </View>
      ) : overview.isError ? (
        <LoadError error={overview.error} onRetry={() => void overview.refetch()} />
      ) : (
        <ScrollView className="flex-1 px-n4 pt-n4" contentContainerClassName="gap-n3 pb-n6">
          <Kicker className="ml-[2px]">{t("household.members")}</Kicker>

          {members.length === 0 ? (
            <EmptyState
              icon="users-three"
              title={t("household.members")}
              action={{ label: t("household.invite"), onPress: openInvite }}
            />
          ) : (
            members.map((entry, index) => (
              <MemberCard
                key={entry.member?.userId ?? String(index)}
                index={index}
                name={entry.member?.displayName ?? ""}
                owner={entry.member?.role === MemberRole.OWNER}
                pending={entry.member?.status === MemberStatus.PENDING}
                privateAccountCount={entry.privateAccountCount}
                spent={fromWire(entry.spent, currency)}
                share={entry.share}
                transactionCount={entry.transactionCount}
                onPress={
                  entry.member?.userId
                    ? () =>
                        router.push({
                          pathname: "/(app)/members/spending",
                          params: { memberId: entry.member?.userId ?? "" },
                        })
                    : undefined
                }
              />
            ))
          )}

          {members.length > 0 ? (
            <Button
              title={t("household.invite")}
              icon="user-plus"
              variant="ghost"
              onPress={openInvite}
              className="border-accent"
            />
          ) : null}

          <ListSection title={t("household.shared")} className="mt-n3">
            <Row
              title={t("household.sharedAccounts")}
              leading={<Icon name="wallet" size={18} color={nocturne.accent[400]} />}
              trailing={<Count value={data?.sharedAccountCount ?? 0} />}
              chevron
              onPress={() => router.push("/(app)/accounts")}
            />
            <Row
              title={t("household.groupBudgets")}
              leading={<Icon name="target" size={18} color={nocturne.accent[400]} />}
              trailing={<Count value={data?.groupBudgetCount ?? 0} />}
              chevron
              onPress={() => router.push("/(app)/categories")}
            />
            <ToggleRow
              label={t("household.overspendNotifications")}
              value={overspendOn}
              onValueChange={toggleOverspend}
              divider={false}
            />
          </ListSection>
        </ScrollView>
      )}

      <BottomIndicator />

      <Sheet visible={inviteOpen} onClose={() => setInviteOpen(false)} title={t("household.invite")}>
        <Field
          label={t("auth.email")}
          value={inviteEmail}
          onChangeText={setInviteEmail}
          autoCapitalize="none"
          autoCorrect={false}
          keyboardType="email-address"
          error={inviteError ?? undefined}
        />
        <Button
          title={t("household.invite")}
          onPress={() => void submitInvite()}
          disabled={invite.isPending || inviteEmail.trim() === ""}
          className="mt-n4"
        />
      </Sheet>

      <Drawer
        visible={menuOpen}
        onClose={() => setMenuOpen(false)}
        account={{ name: familyName, email: "" }}
        household={{ name: familyName, balance: sharedBalance }}
        items={drawerItems}
        activeId="household"
        onSelect={(item) => {
          setMenuOpen(false);
          if (item.href && item.id !== "household") router.push(item.href as "/(app)");
        }}
      />
    </Screen>
  );
}

/** One member: who they are, what they may see, and their share of the period. */
function MemberCard({
  index,
  name,
  owner,
  pending,
  privateAccountCount,
  spent,
  share,
  transactionCount,
  onPress,
}: {
  index: number;
  name: string;
  owner: boolean;
  pending: boolean;
  privateAccountCount: number;
  spent: Money;
  share: number;
  transactionCount: number;
  onPress?: () => void;
}) {
  const { t } = useI18n();
  const access = pending ? t("onboarding.pending") : t("household.fullAccess");
  const meta = t("household.accessMeta", {
    access,
    accounts: t("common.privateAccountCount", { count: privateAccountCount }),
  });

  return (
    <Card
      padded={false}
      onPress={onPress}
      accessibilityLabel={name}
      className={`p-n4 border ${owner ? "border-neutral-700" : "border-neutral-800"}`}
    >
      <View className="flex-row items-center gap-n3">
        <MemberAvatar name={name} index={index} size={40} />
        <View className="flex-1">
          <View className="flex-row items-center gap-n2">
            <Text className="text-[14.5px] font-medium text-fg" numberOfLines={1}>
              {name}
            </Text>
            {owner ? <Badge label={t("household.owner")} tone="accent" /> : null}
          </View>
          <Text className="mt-[2px] text-[11px] text-neutral-600" numberOfLines={1}>
            {meta}
          </Text>
        </View>
        {onPress ? <Icon name="caret-right" size={14} color={nocturne.neutral[600]} /> : null}
      </View>

      <Divider className="mt-n4" />
      <View className="mt-n4 flex-row gap-n4">
        <View className="flex-1">
          <Text className="text-[10.5px] uppercase text-neutral-600" style={{ letterSpacing: 0.8 }}>
            {t("household.spending")}
          </Text>
          <MoneyText value={spent} size={15} className="mt-[3px]" />
        </View>
        <Stat label={t("household.share")} value={formatPercent(share)} />
        <Stat label={t("household.transactions")} value={String(transactionCount)} />
      </View>
    </Card>
  );
}

/** The small grey counter the shared rows carry — "5", "4". */
function Count({ value }: { value: number }) {
  return <Text className="text-[12px] text-neutral-600">{String(value)}</Text>;
}

/** Same shape the app gate uses, so a failed overview reads like every other failure. */
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
