import {
  toDisplayError,
  useBootstrapHousehold,
  useCreateFamily,
  useFamily,
  useInviteMember,
} from "@fm/api";
import { useAuth } from "@fm/auth";
import { useRouter } from "expo-router";
import { getCalendars } from "expo-localization";
import { useState } from "react";
import { KeyboardAvoidingView, Platform, Pressable, ScrollView, Text, View } from "react-native";

import { useI18n } from "@/components/i18n";
import {
  Badge,
  Button,
  Card,
  Field,
  Icon,
  Kicker,
  MemberAvatar,
  Row,
  Screen,
  Sheet,
  nocturne,
} from "@/components/nocturne";

/**
 * Screen 01 — create the household.
 *
 * The only screen that runs before a family exists, so it is the one place the gate lets
 * through on `FailedPrecondition`. Two shapes, one layout:
 *
 *  - no family yet — the name is typed, invitees are collected locally, and "Далі" performs
 *    the whole sequence: CreateFamily → token refresh (the access token carries family_id, and
 *    BootstrapHousehold reads it from the claims) → InviteMember per collected address →
 *    BootstrapHousehold → home;
 *  - a family already exists (the user reopened this route) — the members come from the
 *    service, an invite goes out immediately, and "Далі" only bootstraps and leaves.
 */

/** The household's base currency. The design is a UAH household; changing it is a settings
 * decision (screen 10), not an onboarding question. */
const BASE_CURRENCY = "UAH";
const FALLBACK_TIMEZONE = "Europe/Kyiv";

export default function OnboardingScreen() {
  const { t } = useI18n();
  const router = useRouter();
  const { refreshNow } = useAuth();

  const family = useFamily();
  const createFamily = useCreateFamily();
  const inviteMember = useInviteMember();
  const bootstrap = useBootstrapHousehold();

  const existing = family.data?.family ?? null;
  const members = family.data?.members ?? [];

  const [name, setName] = useState(existing?.name ?? "");
  const [invites, setInvites] = useState<string[]>([]);
  const [inviteOpen, setInviteOpen] = useState(false);
  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteError, setInviteError] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [errorRef, setErrorRef] = useState<string | null>(null);
  const [nameError, setNameError] = useState<string | null>(null);

  const busy = createFamily.isPending || inviteMember.isPending || bootstrap.isPending;
  const typedName = name.trim();

  /** Before the family exists an invite has nowhere to go, so it waits in local state and is
   * sent as part of "Далі". Afterwards it goes out at once. */
  const addInvite = async () => {
    const email = inviteEmail.trim();
    if (email === "") return;
    setInviteError(null);
    if (existing) {
      try {
        await inviteMember.mutateAsync({ familyId: existing.id, email });
      } catch (e) {
        setInviteError(toDisplayError(e, t("household.inviteError")).message);
        return;
      }
    }
    setInvites((current) => (current.includes(email) ? current : [...current, email]));
    setInviteEmail("");
    setInviteOpen(false);
  };

  const submit = async () => {
    setError(null);
    setErrorRef(null);
    setNameError(null);

    if (!existing && typedName === "") {
      setNameError(t("onboarding.nameRequired"));
      return;
    }

    try {
      let familyId = existing?.id ?? "";
      if (!existing) {
        const created = await createFamily.mutateAsync(typedName);
        familyId = created.family?.id ?? "";
        // The access token carries family_id; BootstrapHousehold reads it from the claims,
        // so the session has to pick up the new one before the next call.
        await refreshNow();
        for (const email of invites) {
          await inviteMember.mutateAsync({ familyId, email });
        }
      }

      await bootstrap.mutateAsync({
        baseCurrencyCode: BASE_CURRENCY,
        timezone: getCalendars()[0]?.timeZone ?? FALLBACK_TIMEZONE,
      });
      await family.refetch();
      router.replace("/(app)");
    } catch (e) {
      const shown = toDisplayError(e, t("onboarding.createError"));
      setError(shown.message);
      setErrorRef(shown.reference ?? null);
    }
  };

  return (
    <Screen>
      <KeyboardAvoidingView className="flex-1" behavior={Platform.OS === "ios" ? "padding" : undefined}>
        <ScrollView
          className="flex-1"
          contentContainerStyle={{ flexGrow: 1, paddingHorizontal: nocturne.space.n6, paddingTop: 40, paddingBottom: nocturne.space.n5 }}
          keyboardShouldPersistTaps="handled"
        >
          <View className="mb-n6 h-[44px] w-[44px] items-center justify-center rounded-md border border-accent">
            <Icon name="users-three" size={22} color={nocturne.accent[400]} />
          </View>

          <Text className="text-[27px] font-medium leading-[31px] text-fg">{t("onboarding.title")}</Text>
          <Text className="mb-n6 mt-n3 text-[13.5px] leading-[21px] text-neutral-500">
            {t("onboarding.body")}
          </Text>

          <Field
            label={t("onboarding.nameLabel")}
            placeholder={t("onboarding.namePlaceholder")}
            value={name}
            onChangeText={(next) => {
              setName(next);
              if (nameError) setNameError(null);
            }}
            editable={!existing && !busy}
            error={nameError ?? undefined}
          />

          <Kicker className="mb-n3 mt-n6">{t("onboarding.members")}</Kicker>
          <View className="gap-n3">
            {members.map((member, index) => (
              <Card key={member.userId} padded={false}>
                <Row
                  title={member.displayName}
                  subtitle={member.email}
                  leading={<MemberAvatar name={member.displayName} index={index} size={32} />}
                  trailing={
                    existing && member.userId === existing.ownerUserId ? (
                      <Badge label={t("onboarding.owner")} tone="accent" />
                    ) : undefined
                  }
                  divider={false}
                />
              </Card>
            ))}

            {invites.map((email, index) => (
              <Card key={email} padded={false}>
                <Row
                  title={email}
                  subtitle={t("onboarding.pending")}
                  leading={<MemberAvatar name={email} index={members.length + index} size={32} />}
                  trailing={<Icon name="clock" size={15} color={nocturne.neutral[600]} />}
                  divider={false}
                />
              </Card>
            ))}
          </View>

          <InviteAction label={t("onboarding.inviteByLink")} onPress={() => setInviteOpen(true)} disabled={busy} />

          <View className="flex-1" />

          {error ? (
            <Text className="mb-n3 text-[12.5px] leading-[19px] text-overspend">{error}</Text>
          ) : null}
          {errorRef ? (
            <Text className="mb-n3 text-[12px] text-neutral-600">
              {t("common.errorReference", { ref: errorRef })}
            </Text>
          ) : null}

          <Button
            title={busy ? t("common.loadingEllipsis") : t("onboarding.next")}
            icon="arrow-right"
            variant="ghost"
            className="mt-n5 border-accent"
            disabled={busy}
            onPress={() => void submit()}
          />

          <StepDots count={3} active={0} />
        </ScrollView>
      </KeyboardAvoidingView>

      <Sheet
        visible={inviteOpen}
        onClose={() => {
          setInviteOpen(false);
          setInviteError(null);
        }}
        title={t("onboarding.inviteByLink")}
      >
        <Field
          label={t("auth.email")}
          value={inviteEmail}
          onChangeText={setInviteEmail}
          autoCapitalize="none"
          keyboardType="email-address"
          autoComplete="email"
          error={inviteError ?? undefined}
        />
        <Button
          title={t("common.add")}
          className="mt-n5"
          disabled={inviteEmail.trim() === "" || inviteMember.isPending}
          onPress={() => void addInvite()}
        />
      </Sheet>
    </Screen>
  );
}

/** The dashed full-width invite affordance. The kit's dashed variant is a Chip — a pill sized
 * to its label — so this row is drawn here rather than bent out of one. */
function InviteAction({ label, onPress, disabled }: { label: string; onPress: () => void; disabled: boolean }) {
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ disabled }}
      disabled={disabled}
      onPress={onPress}
      className="mt-n4 flex-row items-center justify-center gap-n2 rounded-md border border-dashed border-neutral-700 px-n5 py-n4"
      style={({ pressed }) => (pressed ? { opacity: 0.75 } : null)}
    >
      <Icon name="user-plus" size={17} color={nocturne.neutral[400]} />
      <Text className="text-[13px] font-medium text-neutral-400">{label}</Text>
    </Pressable>
  );
}

/** Onboarding's progress dots. Decoration — the flow's other two steps are elsewhere. */
function StepDots({ count, active }: { count: number; active: number }) {
  return (
    <View className="mt-n4 flex-row justify-center gap-[5px]" pointerEvents="none">
      {Array.from({ length: count }, (_, index) => (
        <View
          key={index}
          className="h-[3px] rounded-sm"
          style={{
            width: index === active ? 18 : 6,
            backgroundColor: index === active ? nocturne.accent.DEFAULT : nocturne.neutral[800],
          }}
        />
      ))}
    </View>
  );
}
