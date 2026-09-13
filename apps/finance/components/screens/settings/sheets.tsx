import { fromWire, type Account, type HouseholdFinanceSettings } from "@fm/api";
import { Text, View } from "react-native";

import { LOCALES, useI18n, type Locale, type TranslationKey } from "@/components/i18n";
import {
  Button,
  Kicker,
  MoneyText,
  Row,
  SegmentedTabs,
  Sheet,
  ToggleRow,
} from "@/components/nocturne";


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
    <Sheet visible={visible} onClose={onClose} title={t("settings.privacy")} scroll>
      {privateOwn.length === 0 ? (
        <Text className="py-n5 text-[13px] text-neutral-500">{t("common.none")}</Text>
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
    </Sheet>
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
    <Sheet visible={visible} onClose={onClose} title={t("settings.pin")}>
      <View className="overflow-hidden rounded-lg bg-bg">
        <ToggleRow label={t("settings.pin")} value={enabled} onValueChange={onChange} divider={false} />
      </View>
    </Sheet>
  );
}

export function AppearanceSheet({ visible, onClose }: { visible: boolean; onClose: () => void }) {
  const { t, locale, setLocale } = useI18n();
  return (
    <Sheet visible={visible} onClose={onClose} title={t("settings.appearance")}>
      <View className="gap-n3 pb-n4">
        <Kicker>{t("settings.language")}</Kicker>
        <SegmentedTabs<Locale>
          options={LOCALES.map((value) => ({ value, label: value.toUpperCase() }))}
          value={locale}
          onChange={setLocale}
        />
      </View>
    </Sheet>
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
    <Sheet visible={visible} onClose={onClose} title={t("settings.data")}>
      <View className="gap-n3 pb-n4">
        <Text className="text-[13px] text-neutral-300">{facts}</Text>
        <Text className="text-[11.5px] text-neutral-600">{syncedAt}</Text>
      </View>
    </Sheet>
  );
}

export function AdvancedSheet({
  visible,
  onClose,
  version,
  onSignOut,
}: {
  visible: boolean;
  onClose: () => void;
  version: string;
  onSignOut: () => void;
}) {
  const { t } = useI18n();
  return (
    <Sheet visible={visible} onClose={onClose} title={t("settings.advanced")}>
      <View className="gap-n4 pb-n4">
        <Text className="text-[11.5px] text-neutral-600">{version}</Text>
        <Button title={t("settings.signOut")} variant="ghost" onPress={onSignOut} />
      </View>
    </Sheet>
  );
}
