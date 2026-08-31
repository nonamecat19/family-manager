import { AccountKind, AccountVisibility, money, toDisplayError, useCreateAccount } from "@fm/api";
import { useState } from "react";
import { View } from "react-native";

import { useI18n } from "@/components/i18n";
import { Button, Field, Sheet, ToggleRow } from "@/components/nocturne";

/**
 * The Fab's sheet: name plus the one decision that changes what the rest of the family sees.
 *
 * Deliberately narrow. Kind, currency and opening balance have no copy in the dictionary and
 * no cell in the design canvas, so this creates a cash account in the household's currency
 * with a zero opening balance and leaves the rest to the account editor. Private implies
 * excluded from the family total — that is the contract's own rule, not a choice offered here.
 */
export interface NewAccountSheetProps {
  visible: boolean;
  onClose: () => void;
  /** The household's headline currency, so a new account lands in the one the family counts in. */
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
    <Sheet visible={visible} onClose={close} title={t("accounts.addAccount")}>
      <View className="gap-n4 pb-n4">
        <Field
          label={t("accounts.addAccount")}
          value={name}
          onChangeText={setName}
          autoFocus
          error={error ?? undefined}
        />

        <ToggleRow
          label={t("accounts.visibleToAll")}
          subtitle={shared ? undefined : t("accounts.excluded")}
          value={shared}
          onValueChange={setShared}
          divider={false}
        />

        <View className="flex-row gap-n3">
          <Button title={t("common.cancel")} variant="ghost" onPress={close} className="flex-1" />
          <Button
            title={t("common.save")}
            disabled={name.trim() === "" || create.isPending}
            onPress={() => void submit()}
            className="flex-1"
          />
        </View>
      </View>
    </Sheet>
  );
}
