import { useRouter } from "expo-router";
import { Pressable, ScrollView, Text, View } from "react-native";

import { useI18n, type Locale } from "../../components/i18n/index.tsx";
import { CheckIcon } from "../../components/organic/icons.tsx";
import { organic } from "../../components/organic/tokens.ts";
import { Display, Kicker, RoundButton, Screen } from "../../components/organic/ui.tsx";

export default function PreferencesScreen() {
  const router = useRouter();
  const { t, locale, setLocale } = useI18n();

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
      </ScrollView>
    </Screen>
  );
}
