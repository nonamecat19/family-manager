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
import { Button, Chip, Field, Icon, Kicker, Sheet, nocturne } from "@/components/nocturne";

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
    <Sheet visible={visible} onClose={close} title={t("accounts.transfer")}>
      <View className="gap-n4 pb-n4">
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
          <Icon name="arrows-left-right" size={18} color={nocturne.accent[400]} />
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

        <View className="flex-row gap-n3">
          <Button title={t("common.cancel")} variant="ghost" onPress={close} className="flex-1" />
          <Button
            title={t("common.save")}
            disabled={!canSubmit}
            onPress={() => void submit()}
            className="flex-1"
          />
        </View>
      </View>
    </Sheet>
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
      <Kicker className="mb-n2">{label}</Kicker>
      <ScrollView
        horizontal
        showsHorizontalScrollIndicator={false}
        contentContainerStyle={{ gap: nocturne.space.n2 }}
      >
        {accounts.map((account) => (
          <Chip
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
