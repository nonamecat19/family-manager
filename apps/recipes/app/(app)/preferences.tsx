import { toDisplayError, useTelegramLink } from "@fm/api";
import Constants from "expo-constants";
import * as Linking from "expo-linking";
import { useRouter } from "expo-router";
import { Pressable, ScrollView, Text, View } from "react-native";

import { useI18n, type Locale } from "../../components/i18n/index.tsx";
import { CheckIcon, Icon } from "../../components/organic/icons.tsx";
import { NotificationsSection } from "../../components/organic/NotificationsSection.tsx";
import { organic } from "../../components/organic/tokens.ts";
import { Display, Kicker, PrimaryButton, RoundButton, Screen } from "../../components/organic/ui.tsx";

const telegramBot = (Constants.expoConfig?.extra as { telegramBot?: string } | undefined)?.telegramBot;

export default function PreferencesScreen() {
  const router = useRouter();
  const { t, locale, setLocale } = useI18n();
  const telegram = useTelegramLink({ bot: telegramBot, open: Linking.openURL });
  const telegramError = telegram.error
    ? toDisplayError(telegram.error, t("preferences.telegramFailed")).message
    : null;

  const options: { value: Locale; label: string }[] = [
    { value: "en", label: t("preferences.english") },
    { value: "uk", label: t("preferences.ukrainian") },
  ];

  return (
    <Screen>
      <ScrollView showsVerticalScrollIndicator={false} contentContainerClassName="gap-[20px] px-[22px] pb-[28px] pt-[8px]">
        <View className="flex-row items-center gap-[12px]">
          <RoundButton icon="back" label={t("common.back")} onPress={() => router.back()} />
          <Display size={28}>{t("preferences.title")}</Display>
        </View>

        <View>
          <Kicker className="mb-[11px]">{t("preferences.language")}</Kicker>
          <Text className="mb-[12px] font-fig text-[14px] leading-[21px] text-neutral-600">
            {t("preferences.languageHint")}
          </Text>
          <View className="rounded-2xl bg-neutral-100 px-[16px] py-[4px]">
            {options.map((option, i) => {
              const active = option.value === locale;
              return (
                <Pressable
                  key={option.value}
                  accessibilityRole="button"
                  accessibilityLabel={option.label}
                  accessibilityState={{ selected: active }}
                  onPress={() => setLocale(option.value)}
                  className={`flex-row items-center justify-between py-[14px] ${
                    i === options.length - 1 ? "" : "border-b border-divider"
                  }`}
                >
                  <Text className="font-fig-bold text-[15.5px] text-fg">{option.label}</Text>
                  {active && (
                    <View
                      className="h-[22px] w-[22px] items-center justify-center rounded-full"
                      style={{ backgroundColor: organic.accent.DEFAULT }}
                    >
                      <CheckIcon size={12} />
                    </View>
                  )}
                </Pressable>
              );
            })}
          </View>
        </View>

        {telegram.available && (
          <View>
            <Kicker className="mb-[11px]">{t("preferences.telegram")}</Kicker>
            <Text className="mb-[12px] font-fig text-[14px] leading-[21px] text-neutral-600">
              {telegram.awaiting ? t("preferences.telegramWaiting") : t("preferences.telegramHint")}
            </Text>
            <View className="mb-[12px] rounded-2xl bg-neutral-100 px-[16px] py-[14px]">
              <Text className="font-fig-bold text-[15.5px] text-fg">
                {telegram.identity ? t("preferences.telegramConnected") : t("preferences.telegramNotConnected")}
              </Text>
            </View>
            {telegramError && (
              <Text className="mb-[12px] font-fig-semi text-[12px]" style={{ color: organic.danger }}>
                {telegramError}
              </Text>
            )}
            {telegram.identity ? (
              <PrimaryButton
                title={t("preferences.telegramDisconnect")}
                disabled={telegram.unlinking}
                onPress={() => void telegram.unlink().catch(() => undefined)}
              />
            ) : (
              <PrimaryButton
                title={t("preferences.telegramConnect")}
                disabled={telegram.connecting || telegram.loading}
                onPress={() => void telegram.connect().catch(() => undefined)}
              />
            )}
          </View>
        )}

        <View>
          <Kicker className="mb-[11px]">{t("preferences.connectedAccounts")}</Kicker>
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={t("preferences.connectedAccounts")}
            onPress={() => router.push("/(app)/connected-accounts")}
            className="flex-row items-center justify-between rounded-2xl bg-neutral-100 px-[16px] py-[14px]"
          >
            <Text className="font-fig-bold text-[15.5px] text-fg">
              {t("preferences.connectedAccounts")}
            </Text>
            <Icon name="forward" size={16} color={organic.neutral[600]} />
          </Pressable>
        </View>

        <NotificationsSection />

        <View>
          <Kicker className="mb-[11px]">{t("preferences.approveDevice")}</Kicker>
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={t("preferences.approveDevice")}
            onPress={() => router.push("/(app)/approve-device")}
            className="flex-row items-center justify-between rounded-2xl bg-neutral-100 px-[16px] py-[14px]"
          >
            <Text className="font-fig-bold text-[15.5px] text-fg">
              {t("preferences.approveDevice")}
            </Text>
            <Icon name="forward" size={16} color={organic.neutral[600]} />
          </Pressable>
        </View>
      </ScrollView>
    </Screen>
  );
}
