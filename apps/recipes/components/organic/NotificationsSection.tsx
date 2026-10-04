import { useNotificationPreferences, useSetNotificationPreferences } from "@fm/api";
import { Switch, Text, View } from "react-native";

import { useI18n } from "../i18n/index.tsx";
import { organic } from "./tokens.ts";
import { Kicker } from "./ui.tsx";

export function NotificationsSection() {
  const { t } = useI18n();
  const prefs = useNotificationPreferences();
  const setPrefs = useSetNotificationPreferences();

  const muted = prefs.data?.muted ?? [];
  const topics = prefs.data?.topics ?? [];
  const domains = [...new Set(topics.map((topic) => topic.domain))];

  const toggle = (key: string, muteIt: boolean) => {
    const next = new Set(muted);
    if (muteIt) next.add(key);
    else next.delete(key);
    setPrefs.mutate([...next]);
  };

  return (
    <View>
      <Kicker className="mb-[11px]">{t("preferences.notifications")}</Kicker>
      {prefs.isError ? (
        <Text className="font-fig text-[13px] text-neutral-600">
          {t("preferences.notificationsFailed")}
        </Text>
      ) : (
        <View className="rounded-2xl bg-neutral-100 px-[16px] py-[4px]">
          {domains.map((domain) => (
            <View key={domain}>
              <ToggleItem
                label={domain}
                enabled={!muted.includes(domain)}
                onToggle={(enabled) => toggle(domain, !enabled)}
              />
              {topics
                .filter((topic) => topic.domain === domain)
                .map((topic) => (
                  <View key={topic.key} className="pl-[16px]">
                    <ToggleItem
                      label={topic.key}
                      enabled={!muted.includes(domain) && !muted.includes(topic.key)}
                      disabled={muted.includes(domain)}
                      onToggle={(enabled) => toggle(topic.key, !enabled)}
                    />
                  </View>
                ))}
            </View>
          ))}
        </View>
      )}
    </View>
  );
}

function ToggleItem({
  label,
  enabled,
  disabled,
  onToggle,
}: {
  label: string;
  enabled: boolean;
  disabled?: boolean;
  onToggle: (enabled: boolean) => void;
}) {
  return (
    <View className="flex-row items-center justify-between py-[14px]">
      <Text className="flex-1 font-fig-bold text-[15px] text-fg" numberOfLines={1}>
        {label}
      </Text>
      <Switch
        value={enabled}
        onValueChange={onToggle}
        disabled={disabled}
        trackColor={{ false: organic.neutral[300], true: organic.accent.DEFAULT }}
        thumbColor={organic.neutral[100]}
      />
    </View>
  );
}
