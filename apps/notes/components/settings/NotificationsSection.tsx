import { useNotificationPreferences, useSetNotificationPreferences } from "@fm/api";
import { Switch, Text, View } from "react-native";

import { strings } from "../i18n/index.ts";
import { nocturne } from "../nocturne/index.ts";

export function NotificationsSection() {
  const prefs = useNotificationPreferences();
  const setPrefs = useSetNotificationPreferences();

  if (prefs.isPending) return null;
  if (prefs.isError) {
    return <Text className="font-sans text-[13px] text-neutral-400">{strings.settings.notificationsLoadFailed}</Text>;
  }

  const muted = prefs.data.muted;
  const topics = prefs.data.topics;
  const domains = [...new Set(topics.map((topic) => topic.domain))];

  const toggle = (key: string, muteIt: boolean) => {
    const next = new Set(muted);
    if (muteIt) next.add(key);
    else next.delete(key);
    setPrefs.mutate([...next]);
  };

  return (
    <View className="gap-[14px]">
      {domains.map((domain) => (
        <View key={domain} className="gap-[8px]">
          <ToggleRow
            label={domain}
            enabled={!muted.includes(domain)}
            onToggle={(enabled) => toggle(domain, !enabled)}
          />
          {topics
            .filter((topic) => topic.domain === domain)
            .map((topic) => (
              <View key={topic.key} className="pl-[16px]">
                <ToggleRow
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
  );
}

function ToggleRow({
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
    <View className="flex-row items-center gap-[10px]">
      <Text className="flex-1 font-sans text-[14px] text-fg" numberOfLines={1}>
        {label}
      </Text>
      <Switch
        value={enabled}
        onValueChange={onToggle}
        disabled={disabled}
        trackColor={{ false: nocturne.neutral[700], true: nocturne.accent.DEFAULT }}
        thumbColor={nocturne.neutral[100]}
      />
    </View>
  );
}
