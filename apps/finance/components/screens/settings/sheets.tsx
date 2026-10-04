import { fromWire, type Account, type HouseholdFinanceSettings } from "@fm/api";
import { Text, View } from "react-native";

import { useI18n, type TranslationKey } from "@/components/i18n";
import { MoneyText, Row, ScrollSheet } from "@/components/kit";
import { SettingsToggleRow } from "@fm/ui";

export function PrivacySheet({
  visible,
  onClose,
  privateOwn,
  currencyCode,
}: {
  visible: boolean;
  onClose: () => void;
  privateOwn: readonly Account[];
  currencyCode: string;
}) {
  const { t } = useI18n();
  return (
    <ScrollSheet visible={visible} onClose={onClose} title={t("settings.privacy")} scroll>
      {privateOwn.length === 0 ? (
        <Text className="py-[16.8px] text-[13px] text-neutral-600">{t("common.none")}</Text>
      ) : (
        <View className="overflow-hidden rounded-lg bg-bg">
          {privateOwn.map((account, index) => (
            <Row
              key={account.id}
              title={account.name}
              divider={index < privateOwn.length - 1}
              trailing={<MoneyText value={fromWire(account.balance, currencyCode)} size={14} />}
            />
          ))}
        </View>
      )}
    </ScrollSheet>
  );
}

export function PinSheet({
  visible,
  onClose,
  enabled,
  onChange,
}: {
  visible: boolean;
  onClose: () => void;
  enabled: boolean;
  onChange: (next: boolean) => void;
}) {
  const { t } = useI18n();
  return (
    <ScrollSheet visible={visible} onClose={onClose} title={t("settings.pin")}>
      <View className="overflow-hidden rounded-lg bg-bg">
        <SettingsToggleRow label={t("settings.pin")} value={enabled} onValueChange={onChange} divider={false} />
      </View>
    </ScrollSheet>
  );
}

export function DataSheet({
  visible,
  onClose,
  settings,
  syncedAt,
}: {
  visible: boolean;
  onClose: () => void;
  settings: HouseholdFinanceSettings | null;
  syncedAt: string;
}) {
  const { t } = useI18n();
  const weekday = settings?.weekStartsOn
    ? t(`common.weekdayShort.${settings.weekStartsOn}` as TranslationKey)
    : "";
  const facts = settings
    ? [settings.baseCurrencyCode, settings.timezone, weekday].filter((fact) => fact !== "").join(" · ")
    : "";
  return (
    <ScrollSheet visible={visible} onClose={onClose} title={t("settings.data")}>
      <View className="gap-[8.4px] pb-[11.2px]">
        <Text className="text-[13px] text-neutral-600">{facts}</Text>
        <Text className="text-[11.5px] text-neutral-600">{syncedAt}</Text>
      </View>
    </ScrollSheet>
  );
}

export function AdvancedSheet({
  visible,
  onClose,
  version,
}: {
  visible: boolean;
  onClose: () => void;
  version: string;
}) {
  const { t } = useI18n();
  return (
    <ScrollSheet visible={visible} onClose={onClose} title={t("settings.advanced")}>
      <View className="gap-[11.2px] pb-[11.2px]">
        <Text className="text-[11.5px] text-neutral-600">{version}</Text>
      </View>
    </ScrollSheet>
  );
}
