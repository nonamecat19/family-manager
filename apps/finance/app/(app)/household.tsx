import { LoadError } from "@/components/kit/LoadError.tsx";
import {
  currentPeriod,
  FamilyMemberRole,
  familyStanding,
  fromWire,
  InvitationStatus,
  MemberRole,
  MemberStatus,
  toDisplayError,
  useFamily,
  useFinanceSettings,
  useHouseholdOverview,
  useInvitations,
  useInviteMember,
  useLeaveFamily,
  useRemoveMember,
  useRevokeInvitation,
  useSetOverspendNotifications,
  useUpdateFamily,
  type Money,
} from "@fm/api";
import { useAuth } from "@fm/auth";
import { FamilyInvitationsCard, FamilyMembersCard, FamilyNameCard, LeaveFamilyCard, type FamilyInvitationView, type FamilyMemberView, Badge, Button, Divider, EmptyState, Field, Icon, IconButton, Kicker, Avatar, organic, Screen, ScreenHeader, ScrollBody, Sheet, SettingsToggleRow } from "@fm/ui";
import { useRouter } from "expo-router";
import { useMemo, useState } from "react";
import { Text, View } from "react-native";

import { useI18n } from "@/components/i18n";
import { Card, formatPercent, ListSection, MoneyText, monthNameLower, parseISO, Row, Stat } from "@/components/kit";

export default function HouseholdScreen() {
  const { t } = useI18n();
  const router = useRouter();

  const period = useMemo(() => currentPeriod("month"), []);
  const anchorMonth = "anchor" in period ? parseISO(period.anchor).month : new Date().getMonth() + 1;

  const overview = useHouseholdOverview(period);
  const family = useFamily();
  const settings = useFinanceSettings();
  const invite = useInviteMember();
  const setOverspend = useSetOverspendNotifications();

  const { claims, refreshNow } = useAuth();
  const familyId = family.data?.family?.id ?? "";
  const familyMembers = useMemo(() => family.data?.members ?? [], [family.data]);
  const { isAdmin, canLeave } = useMemo(
    () => familyStanding(familyMembers, claims?.userId),
    [familyMembers, claims?.userId],
  );

  const invitations = useInvitations(familyId, { enabled: isAdmin });

  const renameFamily = useUpdateFamily();
  const removeMember = useRemoveMember();
  const revokeInvitation = useRevokeInvitation();
  const leaveFamily = useLeaveFamily();

  const [familyError, setFamilyError] = useState<string | null>(null);
  const [removingUserId, setRemovingUserId] = useState<string | null>(null);
  const [revokingId, setRevokingId] = useState<string | null>(null);

  const memberViews = useMemo<FamilyMemberView[]>(
    () =>
      familyMembers.map((member) => ({
        userId: member.userId,
        displayName: member.displayName,
        email: member.email,
        role: member.role === FamilyMemberRole.ADMIN ? "admin" : "member",
        isSelf: member.userId === claims?.userId,
      })),
    [familyMembers, claims?.userId],
  );

  const invitationViews = useMemo<FamilyInvitationView[]>(
    () =>
      (invitations.data?.invitations ?? []).map((invitation) => ({
        id: invitation.id,
        email: invitation.email,
        pending: invitation.status === InvitationStatus.PENDING,
        expiresLabel: invitation.expiresAt
          ? t("family.invitations.expires", {
              date: new Date(Number(invitation.expiresAt.seconds) * 1000).toLocaleDateString(),
            })
          : undefined,
      })),
    [invitations.data, t],
  );

  const submitRename = async (name: string) => {
    setFamilyError(null);
    try {
      await renameFamily.mutateAsync({ familyId, name });
    } catch (error) {
      setFamilyError(toDisplayError(error, t("family.name.error")).message);
    }
  };

  const submitRemove = (member: FamilyMemberView) => {
    setFamilyError(null);
    setRemovingUserId(member.userId);
    removeMember.mutate(
      { familyId, userId: member.userId },
      {
        onError: (error) =>
          setFamilyError(toDisplayError(error, t("family.members.removeError")).message),
        onSettled: () => setRemovingUserId(null),
      },
    );
  };

  const submitRevoke = (invitation: FamilyInvitationView) => {
    setFamilyError(null);
    setRevokingId(invitation.id);
    revokeInvitation.mutate(invitation.id, {
      onError: (error) =>
        setFamilyError(toDisplayError(error, t("family.invitations.revokeError")).message),
      onSettled: () => setRevokingId(null),
    });
  };

  const submitLeave = () => {
    setFamilyError(null);
    leaveFamily.mutate(familyId, {
      onSuccess: () => void refreshNow().then(() => router.replace("/(app)/onboarding")),
      onError: (error) => setFamilyError(toDisplayError(error, t("family.leave.error")).message),
    });
  };

  const [inviteOpen, setInviteOpen] = useState(false);
  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteError, setInviteError] = useState<string | null>(null);
  const [overspendPending, setOverspendPending] = useState<boolean | null>(null);

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
      <ScrollBody>
        <ScreenHeader
          title={familyName}
          onBack={() => router.back()}
          backLabel={t("common.back")}
          actions={
            <IconButton
              icon="gear"
              label={t("nav.settings")}
              onPress={() => router.push("/(app)/settings")}
            />
          }
        />

        <View className="flex-row gap-[20px]">
          <View>
            <Kicker>{t("household.sharedBalance")}</Kicker>
            <MoneyText value={sharedBalance} size={22} className="mt-[2px]" />
          </View>
          <View className="w-[1px]" style={{ backgroundColor: organic.divider }} />
          <View>
            <Kicker>{t("household.spendingIn", { month: monthNameLower(t, anchorMonth) })}</Kicker>
            <MoneyText value={periodExpense} size={22} className="mt-[2px]" />
          </View>
        </View>

        {overview.isPending ? (
          <View className="items-center py-[28px]">
            <Text className="text-[13px] text-neutral-600">{t("common.loadingEllipsis")}</Text>
          </View>
        ) : overview.isError ? (
          <LoadError error={overview.error} onRetry={() => void overview.refetch()} />
        ) : (
          <>
            <Kicker>{t("household.members")}</Kicker>

            {members.length === 0 ? (
              <EmptyState
                title={t("household.members")}
                action={{ label: t("household.invite"), onPress: openInvite }}
              />
            ) : (
              <View className="gap-[8px]">
                {members.map((entry, index) => (
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
                ))}
              </View>
            )}

            {members.length > 0 ? (
              <Button title={t("household.invite")} tone="quiet" onPress={openInvite} />
            ) : null}

            <ListSection title={t("household.shared")}>
              <Row
                title={t("household.sharedAccounts")}
                leading={<Icon name="wallet" size={18} color={organic.accent[600]} />}
                trailing={<Count value={data?.sharedAccountCount ?? 0} />}
                chevron
                onPress={() => router.push("/(app)/accounts")}
              />
              <Row
                title={t("household.groupBudgets")}
                leading={<Icon name="target" size={18} color={organic.accent[600]} />}
                trailing={<Count value={data?.groupBudgetCount ?? 0} />}
                chevron
                onPress={() => router.push("/(app)/categories")}
              />
              <SettingsToggleRow
                label={t("household.overspendNotifications")}
                value={overspendOn}
                onValueChange={toggleOverspend}
                divider={false}
              />
            </ListSection>

            <FamilyNameCard
              name={familyName}
              isAdmin={isAdmin}
              busy={renameFamily.isPending}
              error={familyError}
              onRename={submitRename}
              strings={{
                heading: t("family.name.heading"),
                rename: t("family.name.rename"),
                nameLabel: t("family.name.label"),
                save: t("family.name.save"),
                cancel: t("family.name.cancel"),
                required: t("family.name.required"),
              }}
            />

            <FamilyMembersCard
              members={memberViews}
              isAdmin={isAdmin}
              removingUserId={removingUserId}
              onRemove={submitRemove}
              strings={{
                heading: t("family.members.heading"),
                you: t("family.members.you"),
                admin: t("family.members.admin"),
                remove: t("family.members.remove"),
                empty: t("family.members.empty"),
              }}
            />

            {isAdmin ? (
              <FamilyInvitationsCard
                invitations={invitationViews}
                revokingId={revokingId}
                onRevoke={submitRevoke}
                strings={{
                  heading: t("family.invitations.heading"),
                  revoke: t("family.invitations.revoke"),
                  empty: t("family.invitations.empty"),
                  expired: t("family.invitations.spent"),
                }}
              />
            ) : null}

            <LeaveFamilyCard
              canLeave={canLeave}
              busy={leaveFamily.isPending}
              error={familyError}
              onLeave={submitLeave}
              strings={{
                heading: t("family.leave.heading"),
                body: t("family.leave.body"),
                leave: t("family.leave.action"),
                confirm: t("family.leave.confirm"),
                cancel: t("family.leave.cancel"),
                lastAdmin: t("family.leave.lastAdmin"),
              }}
            />
          </>
        )}
      </ScrollBody>

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
        <View className="mt-[16px]">
          <Button
            title={t("household.invite")}
            onPress={() => void submitInvite()}
            disabled={invite.isPending || inviteEmail.trim() === ""}
          />
        </View>
      </Sheet>
    </Screen>
  );
}

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
      className={`p-[11.2px] border ${owner ? "border-accent" : "border-divider"}`}
    >
      <View className="flex-row items-center gap-[8.4px]">
        <Avatar name={name} index={index} size={40} />
        <View className="flex-1">
          <View className="flex-row items-center gap-[5.6px]">
            <Text className="text-[14.5px] font-fig-med text-fg" numberOfLines={1}>
              {name}
            </Text>
            {owner ? <Badge label={t("household.owner")} tone="accent" /> : null}
          </View>
          <Text className="mt-[2px] text-[11px] text-neutral-600" numberOfLines={1}>
            {meta}
          </Text>
        </View>
        {onPress ? <Icon name="caret-right" size={14} color={organic.neutral[600]} /> : null}
      </View>

      <View className="mt-[11.2px]"><Divider /></View>
      <View className="mt-[11.2px] flex-row gap-[11.2px]">
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

function Count({ value }: { value: number }) {
  return <Text className="text-[12px] text-neutral-600">{String(value)}</Text>;
}
