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

/**
 * The four preference rows on screen 11 draw a caret, so each opens something. None of them
 * has a route in the design's eleven screens, so each opens a sheet on this screen instead of
 * inventing a twelfth route the drawer would then have to list.
 */

/** "Приватність" — the caller's OWN private accounts. Another member's private accounts are
 * never named here: the server sends only a count for those, and this sheet asks for none. */
export function PrivacySheet({
  visible,
  onClose,
  privateOwn,
  currencyCode,
}: {
  visible: boolean;
  onClose: () => void;
  privateOwn: readonly Account[];
  /** The household's base currency, used only when an account carries no amount of its own. */
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

/** "PIN" — `pin_lock_enabled` is a household setting, not a device one, so this writes it. */
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

/** "Вигляд" — Nocturne is dark-only, so the one appearance choice is the language. Locales
 * are labelled by their own code: a language name would be app copy, and screens do not add
 * copy to the dictionary. */
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

/** "Дані та синхронізація" — what the household's data is keyed to, and when this device last
 * heard from the service. */
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

/** "Додатково" — the build, and the one destructive control the app has. */
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
