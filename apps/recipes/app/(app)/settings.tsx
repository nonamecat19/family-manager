import { useFamily, useInviteMember, useMembers, useRecipes } from "@fm/api";
import { useAuth } from "@fm/auth";
import { Role } from "@fm/sdk/family/v1/family_pb";
import { useRouter } from "expo-router";
import { useState } from "react";
import { Pressable, ScrollView, Text, TextInput, View } from "react-native";

import { useI18n } from "../../components/i18n/index.tsx";
import { formatDuration } from "../../components/organic/format.ts";
import { initialOf, organic, tintFor } from "../../components/organic/tokens.ts";
import {
  Avatar,
  DashedButton,
  Display,
  Kicker,
  PrimaryButton,
  Screen,
  Sheet,
} from "../../components/organic/ui.tsx";

/**
 * "You" is the cookbook's colophon: who keeps it, who cooks from it, and how to let one more
 * person in. Account actions live at the bottom because they are the rarest thing here.
 */
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
      <ScrollView showsVerticalScrollIndicator={false} contentContainerClassName="gap-[20px] px-[22px] pb-[28px] pt-[8px]">
        <View className="flex-row items-center gap-[16px]">
          <Avatar
            initial={initialOf(family.data?.family?.name ?? t("profile.familyFallback"))}
            tint={{ bg: organic.accent2[300], fg: organic.accent2[800] }}
            size={72}
          />
          <View className="flex-1">
            <Display size={23}>{family.data?.family?.name ?? t("profile.householdFallback")}</Display>
            <Text className="mt-[4px] font-fig-bold text-[13.5px] text-neutral-600">
              {t("profile.keeperOfTheCookbook")}
            </Text>
          </View>
        </View>

        <View className="flex-row gap-[10px]">
          <Stat value={`${all.length}`} label={t("profile.statRecipes")} />
          <Stat value={`${members.data?.members.length ?? cooks}`} label={t("profile.statCooks")} />
          <Stat value={formatDuration(totalMinutes, t) || "—"} label={t("profile.statTimeWrittenDown")} />
        </View>

        <View>
          <Kicker className="mb-[11px]">{t("profile.family")}</Kicker>
          <View className="rounded-2xl bg-neutral-100 px-[16px] py-[4px]">
            {(members.data?.members ?? []).map((member, i, list) => (
              <View
                key={member.userId}
                className={`flex-row items-center gap-[13px] py-[12px] ${
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
              <Text className="py-[14px] font-fig text-[15px] text-neutral-600">
                {t("profile.justYouSoFar")}
              </Text>
            )}
          </View>
        </View>

        <DashedButton title={t("profile.inviteSomeone")} onPress={() => setInviteOpen(true)} />

        <Pressable
          accessibilityRole="button"
          accessibilityLabel={t("profile.favorites")}
          onPress={() => router.push("/(app)/favorites")}
          className="items-center rounded-full bg-neutral-200 py-[13px]"
        >
          <Text className="font-fig-bold text-[14.5px] text-neutral-700">{t("profile.yourFavourites")}</Text>
        </Pressable>

        <Pressable
          accessibilityRole="button"
          accessibilityLabel={t("profile.preferences")}
          onPress={() => router.push("/(app)/preferences")}
          className="items-center rounded-full bg-neutral-200 py-[13px]"
        >
          <Text className="font-fig-bold text-[14.5px] text-neutral-700">{t("profile.preferences")}</Text>
        </Pressable>

        <Pressable
          accessibilityRole="button"
          accessibilityLabel={status === "authenticated" ? t("profile.signOut") : t("profile.signIn")}
          onPress={() => {
            void signOut();
            router.replace("/(auth)/login");
          }}
          className="items-center pt-[4px]"
        >
          <Text className="font-fig-semi text-[14px]" style={{ color: organic.danger }}>
            {status === "authenticated" ? t("profile.signOut") : t("profile.signIn")}
          </Text>
        </Pressable>
      </ScrollView>

      <Sheet visible={inviteOpen} onClose={() => setInviteOpen(false)} title={t("profile.inviteACook")}>
        <Kicker className="mb-[10px]">{t("profile.theirEmail")}</Kicker>
        <TextInput
          accessibilityLabel={t("profile.email")}
          value={email}
          onChangeText={setEmail}
          autoCapitalize="none"
          keyboardType="email-address"
          placeholder={t("profile.emailPlaceholder")}
          placeholderTextColor={organic.neutral[500]}
          className="mb-[20px] rounded-full border border-neutral-300 bg-neutral-100 px-[16px] py-[12px] font-fig text-[15px] text-fg"
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

function Stat({ value, label }: { value: string; label: string }) {
  return (
    <View className="flex-1 rounded-2xl bg-neutral-100 px-[14px] py-[15px]">
      <Text className="font-cap text-[24px] text-accent-700">{value}</Text>
      <Text className="mt-[3px] font-fig-bold text-[12px] text-neutral-600">{label}</Text>
    </View>
  );
}
