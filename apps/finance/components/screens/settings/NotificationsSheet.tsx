import { useNotificationPreferences, useSetNotificationPreferences } from "@fm/api";
import { Text, View } from "react-native";

import { useI18n } from "@/components/i18n";
import { Sheet, ToggleRow } from "@/components/nocturne";

export function NotificationsSheet({ visible, onClose }: { visible: boolean; onClose: () => void }) {
  const { t } = useI18n();
  const prefs = useNotificationPreferences({ enabled: visible });
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
    <Sheet visible={visible} onClose={onClose} title={t("settings.notifications")} scroll>
      {prefs.isError ? (
        <Text className="py-n4 text-[13px] text-overspend">{t("settings.notificationsFailed")}</Text>
      ) : (
        <View className="overflow-hidden rounded-lg bg-bg">
          {domains.map((domain) => (
            <View key={domain}>
              <ToggleRow
                label={domain}
                value={!muted.includes(domain)}
                onValueChange={(enabled) => toggle(domain, !enabled)}
              />
              {topics
                .filter((topic) => topic.domain === domain)
                .map((topic, index, list) => (
                  <View key={topic.key} className="pl-n5">
                    <ToggleRow
                      label={topic.key}
                      value={!muted.includes(domain) && !muted.includes(topic.key)}
                      onValueChange={(enabled) =>
                        muted.includes(domain) ? undefined : toggle(topic.key, !enabled)
                      }
                      divider={index < list.length - 1}
                    />
                  </View>
                ))}
            </View>
          ))}
        </View>
      )}
    </Sheet>
  );
}
