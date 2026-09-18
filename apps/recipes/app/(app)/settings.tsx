import { useFamily, useInviteMember, useMembers, useRecipes } from "@fm/api";
import { useAuth } from "@fm/auth";
import { Role } from "@fm/sdk/family/v1/family_pb";
import { useRouter } from "expo-router";
import { useState } from "react";
import { ScrollView, Text, TextInput, View } from "react-native";
import { initialOf, organic, tintFor, Avatar, DangerLink, DashedButton, Display, Kicker, PillButton, PrimaryButton, Screen, SettingsGroup, SettingsSection, Sheet, StatTile } from "@fm/ui";

import { useI18n } from "../../components/i18n/index.tsx";
import { formatDuration } from "../../components/organic/format.ts";

export default function ProfileScreen() {
  const router = useRouter();
  const { t } = useI18n();
  const { signOut, status } = useAuth();
  const family = useFamily();
  const familyId = family.data?.family?.id ?? "";
  const members = useMembers(familyId);
  const recipes = useRecipes();

  const invite = useInviteMember();
  const [inviteOpen, setInviteOpen] = useState(false);
  const [email, setEmail] = useState("");

  const all = recipes.data ?? [];
  const cooks = new Set(all.map((r) => r.authorUserId).filter((id) => id !== "")).size;
  const totalMinutes = all.reduce((n, r) => n + r.prepSeconds + r.cookSeconds, 0);

  return (
    <Screen>
      <ScrollView showsVerticalScrollIndicator={false} contentContainerClassName="gap-5 px-5.5 pb-7 pt-sm">
        <View className="flex-row items-center gap-lg">
          <Avatar
            initial={initialOf(family.data?.family?.name ?? t("profile.familyFallback"))}
            tint={{ bg: organic.accent2[300], fg: organic.accent2[800] }}
            size={72}
          />
          <View className="flex-1">
            <Display size={23}>{family.data?.family?.name ?? t("profile.householdFallback")}</Display>
            <Text className="mt-xs font-fig-bold text-[13.5px] text-neutral-600">
              {t("profile.keeperOfTheCookbook")}
            </Text>
          </View>
        </View>

        <View className="flex-row gap-2.5">
          <StatTile value={`${all.length}`} label={t("profile.statRecipes")} />
          <StatTile value={`${members.data?.members.length ?? cooks}`} label={t("profile.statCooks")} />
          <StatTile value={formatDuration(totalMinutes, t) || "—"} label={t("profile.statTimeWrittenDown")} />
        </View>

        <SettingsSection title={t("profile.family")}>
          <SettingsGroup className="py-xs">
            {(members.data?.members ?? []).map((member, i, list) => (
              <View
                key={member.userId}
                className={`flex-row items-center gap-3.25 py-3 ${
                  i === list.length - 1 ? "" : "border-b border-divider"
                }`}
              >
                <Avatar
                  initial={initialOf(member.displayName || member.email)}
                  tint={tintFor(member.userId, i)}
                  size={40}
                />
                <Text className="flex-1 font-fig-bold text-[15px] text-fg" numberOfLines={1}>
                  {member.displayName || member.email}
                </Text>
                <Text className="font-fig-bold text-[12.5px] text-neutral-600">
                  {member.role === Role.ADMIN ? t("profile.owner") : t("profile.cook")}
                </Text>
              </View>
            ))}
            {(members.data?.members ?? []).length === 0 && (
              <Text className="py-3.5 font-fig text-[15px] text-neutral-600">
                {t("profile.justYouSoFar")}
              </Text>
            )}
          </SettingsGroup>
        </SettingsSection>

        <DashedButton title={t("profile.inviteSomeone")} onPress={() => setInviteOpen(true)} />

        <PillButton title={t("profile.yourFavourites")} onPress={() => router.push("/(app)/favorites")} />

        <PillButton title={t("profile.preferences")} onPress={() => router.push("/(app)/preferences")} />

        <DangerLink
          title={status === "authenticated" ? t("profile.signOut") : t("profile.signIn")}
          onPress={() => {
            void signOut();
            router.replace("/(auth)/login");
          }}
        />
      </ScrollView>

      <Sheet visible={inviteOpen} onClose={() => setInviteOpen(false)} title={t("profile.inviteACook")}>
        <Kicker className="mb-2.5">{t("profile.theirEmail")}</Kicker>
        <TextInput
          accessibilityLabel={t("profile.email")}
          value={email}
          onChangeText={setEmail}
          autoCapitalize="none"
          keyboardType="email-address"
          placeholder={t("profile.emailPlaceholder")}
          placeholderTextColor={organic.neutral[500]}
          className="mb-5 rounded-full border border-neutral-300 bg-neutral-100 px-lg py-3 font-fig text-[15px] text-fg"
        />
        <PrimaryButton
          title={invite.isPending ? t("profile.sending") : t("profile.sendTheInvitation")}
          disabled={email.trim() === "" || familyId === "" || invite.isPending}
          onPress={() =>
            invite.mutate(
              { familyId, email: email.trim() },
              {
                onSuccess: () => {
                  setEmail("");
                  setInviteOpen(false);
                },
              },
            )
          }
        />
      </Sheet>
    </Screen>
  );
}
