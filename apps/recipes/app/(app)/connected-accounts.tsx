import { TELEGRAM_PROVIDER, toDisplayError, useIdentities, useUnlinkIdentity } from "@fm/api";
import type { Identity } from "@fm/sdk/auth/v1/auth_pb";
import { useRouter } from "expo-router";
import { useState } from "react";
import { ActivityIndicator, Pressable, ScrollView, Text, View } from "react-native";

import { useI18n, type TranslationKey } from "../../components/i18n/index.tsx";
import { Icon } from "../../components/organic/icons.tsx";
import { organic } from "../../components/organic/tokens.ts";
import { Display, RoundButton, Screen, Sheet } from "../../components/organic/ui.tsx";

function providerLabel(provider: string, t: (key: TranslationKey) => string): string {
  if (provider === TELEGRAM_PROVIDER) return t("connectedAccounts.providerTelegram");
  return provider;
}

function linkedDate(linkedAt: Identity["linkedAt"], locale: string): string {
  if (!linkedAt) return "";
  const date = new Date(Number(linkedAt.seconds) * 1000 + Math.floor(linkedAt.nanos / 1e6));
  return date.toLocaleDateString(locale);
}

export default function ConnectedAccountsScreen() {
  const router = useRouter();
  const { t, locale } = useI18n();
  const identities = useIdentities();
  const unlink = useUnlinkIdentity();
  const [unlinking, setUnlinking] = useState<Identity | null>(null);

  const rows = identities.data ?? [];
  const unlinkError = unlink.isError
    ? toDisplayError(unlink.error, t("connectedAccounts.unlinkFailed")).message
    : null;

  return (
    <Screen>
      <ScrollView
        showsVerticalScrollIndicator={false}
        contentContainerClassName="gap-[18px] px-[22px] pb-[28px] pt-[8px]"
      >
        <View className="flex-row items-center gap-[12px]">
          <RoundButton icon="back" label={t("common.back")} onPress={() => router.back()} />
          <Display size={28}>{t("connectedAccounts.title")}</Display>
        </View>

        {unlinkError ? (
          <Text className="font-fig-semi text-[12px]" style={{ color: organic.danger }}>
            {unlinkError}
          </Text>
        ) : null}

        {identities.isPending ? (
          <View className="items-center py-[28px]">
            <ActivityIndicator color={organic.accent.DEFAULT} />
          </View>
        ) : identities.isError ? (
          <Text className="font-fig text-[14px] leading-[21px] text-neutral-600">
            {toDisplayError(identities.error, t("connectedAccounts.loadFailed")).message}
          </Text>
        ) : rows.length === 0 ? (
          <View className="gap-[8px] py-[18px]">
            <Text className="font-fig-bold text-[15.5px] text-fg">
              {t("connectedAccounts.emptyTitle")}
            </Text>
            <Text className="font-fig text-[14px] leading-[21px] text-neutral-600">
              {t("connectedAccounts.emptyBody")}
            </Text>
          </View>
        ) : (
          <View className="gap-[10px]">
            {rows.map((identity) => (
              <View
                key={`${identity.provider}:${identity.externalId}`}
                className="flex-row items-center gap-[12px] rounded-2xl bg-neutral-100 px-[16px] py-[14px]"
              >
                <View className="flex-1 gap-[3px]">
                  <Text className="font-fig-bold text-[15.5px] text-fg">
                    {providerLabel(identity.provider, t)}
                  </Text>
                  <Text className="font-fig text-[12.5px] text-neutral-600">{identity.externalId}</Text>
                  {identity.linkedAt ? (
                    <Text className="font-fig text-[12px] text-neutral-600">
                      {t("connectedAccounts.linked", { date: linkedDate(identity.linkedAt, locale) })}
                    </Text>
                  ) : null}
                </View>
                <Pressable
                  accessibilityRole="button"
                  accessibilityLabel={t("connectedAccounts.unlink")}
                  disabled={unlink.isPending}
                  onPress={() => setUnlinking(identity)}
                  className="h-[36px] w-[36px] items-center justify-center rounded-full bg-neutral-200"
                >
                  <Icon name="trash" size={16} color={organic.neutral[700]} />
                </Pressable>
              </View>
            ))}
          </View>
        )}
      </ScrollView>

      <Sheet
        visible={unlinking !== null}
        onClose={() => setUnlinking(null)}
        title={t("connectedAccounts.unlinkConfirm")}
      >
        <View className="flex-row gap-[10px] pt-[8px]">
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={t("common.cancel")}
            onPress={() => setUnlinking(null)}
            className="flex-1 items-center justify-center rounded-full border border-divider py-[12px]"
          >
            <Text className="font-fig-bold text-[14px] text-fg">{t("common.cancel")}</Text>
          </Pressable>
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={t("connectedAccounts.unlink")}
            onPress={() => {
              if (!unlinking) return;
              unlink.mutate(
                { provider: unlinking.provider, externalId: unlinking.externalId },
                { onSuccess: () => setUnlinking(null) },
              );
            }}
            className="flex-1 items-center justify-center rounded-full py-[12px]"
            style={{ backgroundColor: organic.danger }}
          >
            <Text className="font-fig-bold text-[14px]" style={{ color: organic.dangerFg }}>
              {t("connectedAccounts.unlink")}
            </Text>
          </Pressable>
        </View>
      </Sheet>
    </Screen>
  );
}
