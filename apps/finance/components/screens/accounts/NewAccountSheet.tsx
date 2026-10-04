import { AccountKind, AccountVisibility, money, toDisplayError, useCreateAccount } from "@fm/api";
import { Button, Field, SettingsToggleRow } from "@fm/ui";
import { useState } from "react";
import { Text, View } from "react-native";

import { useI18n } from "@/components/i18n";
import { ScrollSheet } from "@/components/kit";

export interface NewAccountSheetProps {
  visible: boolean;
  onClose: () => void;
  currencyCode: string;
}

export function NewAccountSheet({ visible, onClose, currencyCode }: NewAccountSheetProps) {
  const { t } = useI18n();
  const create = useCreateAccount();

  const [name, setName] = useState("");
  const [shared, setShared] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const close = () => {
    setName("");
    setShared(true);
    setError(null);
    onClose();
  };

  const submit = async () => {
    const trimmed = name.trim();
    if (trimmed === "") return;
    setError(null);
    try {
      await create.mutateAsync({
        name: trimmed,
        kind: AccountKind.CASH,
        visibility: shared ? AccountVisibility.SHARED : AccountVisibility.PRIVATE,
        currencyCode,
        openingBalance: money(0, currencyCode),
        icon: shared ? "wallet" : "piggy-bank",
        excludedFromFamilyTotal: !shared,
      });
      close();
    } catch (e) {
      setError(toDisplayError(e, t("add.saveError")).message);
    }
  };

  return (
    <ScrollSheet visible={visible} onClose={close} title={t("accounts.addAccount")}>
      <View className="gap-[11.2px] pb-[11.2px]">
        <Field
          label={t("accounts.addAccount")}
          value={name}
          onChangeText={setName}
          autoFocus
          error={error ?? undefined}
        />

        <SettingsToggleRow
          label={t("accounts.visibleToAll")}
          value={shared}
          onValueChange={setShared}
          divider={false}
        />
        {shared ? null : (
          <Text className="font-fig text-[11.5px] text-neutral-600">{t("accounts.excluded")}</Text>
        )}

        <View className="flex-row gap-[8.4px]">
          <Button title={t("common.cancel")} tone="quiet" onPress={close} />
          <Button
            title={t("common.save")}
            disabled={name.trim() === "" || create.isPending}
            onPress={() => void submit()}
          />
        </View>
      </View>
    </ScrollSheet>
  );
}
