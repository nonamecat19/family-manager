import {
  fromWire,
  parseAmount,
  toDisplayError,
  toISODate,
  useTransferBetweenAccounts,
  type Account,
} from "@fm/api";
import { useMemo, useState } from "react";
import { ScrollView, View } from "react-native";

import { useI18n } from "@/components/i18n";
import { AmountChip, ScrollSheet } from "@/components/kit";
import { Button, Field, Icon, Kicker, organic } from "@fm/ui";

export interface TransferSheetProps {
  visible: boolean;
  onClose: () => void;
  accounts: readonly Account[];
}

export function TransferSheet({ visible, onClose, accounts }: TransferSheetProps) {
  const { t } = useI18n();
  const transfer = useTransferBetweenAccounts();

  const [fromId, setFromId] = useState("");
  const [toId, setToId] = useState("");
  const [amount, setAmount] = useState("");
  const [error, setError] = useState<string | null>(null);

  const from = accounts.find((a) => a.id === fromId);
  const currency = from?.currencyCode ?? accounts[0]?.currencyCode ?? "UAH";
  const targets = useMemo(
    () => accounts.filter((a) => a.id !== fromId && a.currencyCode === currency),
    [accounts, fromId, currency],
  );

  const parsed = parseAmount(amount, currency);
  const canSubmit =
    from != null && toId !== "" && parsed != null && parsed.amountMinor > 0 && !transfer.isPending;

  const close = () => {
    setFromId("");
    setToId("");
    setAmount("");
    setError(null);
    onClose();
  };

  const submit = async () => {
    if (!from || !parsed) return;
    setError(null);
    try {
      await transfer.mutateAsync({
        fromAccountId: from.id,
        toAccountId: toId,
        amount: parsed,
        occurredOn: toISODate(new Date()),
      });
      close();
    } catch (e) {
      setError(toDisplayError(e, t("add.saveError")).message);
    }
  };

  return (
    <ScrollSheet visible={visible} onClose={close} title={t("accounts.transfer")}>
      <View className="gap-[11.2px] pb-[11.2px]">
        <AccountPicker
          label={t("add.account")}
          accounts={accounts}
          selectedId={fromId}
          onSelect={(id) => {
            setFromId(id);
            setToId("");
          }}
        />

        <View className="items-center">
          <Icon name="arrows-left-right" size={18} color={organic.accent[600]} />
        </View>

        <AccountPicker
          label={t("add.account")}
          accounts={targets}
          selectedId={toId}
          onSelect={setToId}
        />

        <Field
          label={t("add.amount")}
          value={amount}
          onChangeText={setAmount}
          keyboardType="decimal-pad"
          placeholder="0"
          error={error ?? undefined}
        />

        <View className="flex-row gap-[8.4px]">
          <Button title={t("common.cancel")} tone="quiet" onPress={close} />
          <Button
            title={t("common.save")}
            disabled={!canSubmit}
            onPress={() => void submit()}
          />
        </View>
      </View>
    </ScrollSheet>
  );
}

function AccountPicker({
  label,
  accounts,
  selectedId,
  onSelect,
}: {
  label: string;
  accounts: readonly Account[];
  selectedId: string;
  onSelect: (id: string) => void;
}) {
  return (
    <View>
      <Kicker className="mb-[5.6px]">{label}</Kicker>
      <ScrollView
        horizontal
        showsHorizontalScrollIndicator={false}
        contentContainerStyle={{ gap: 5.6 }}
      >
        {accounts.map((account) => (
          <AmountChip
            key={account.id}
            label={account.name}
            amount={fromWire(account.balance, account.currencyCode)}
            selected={account.id === selectedId}
            onPress={() => onSelect(account.id)}
          />
        ))}
      </ScrollView>
    </View>
  );
}
