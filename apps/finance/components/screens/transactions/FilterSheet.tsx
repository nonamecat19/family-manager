import { View } from "react-native";

import { useI18n } from "@/components/i18n";
import { Button, Chip, Kicker, Sheet } from "@/components/nocturne";

export interface FilterOption {
  id: string;
  label: string;
}

export interface FilterSheetProps {
  visible: boolean;
  onClose: () => void;
  members: readonly FilterOption[];
  /** Shared accounts only — a private account is nobody else's business, and filtering the
   * family feed by one would put its spending back into a family-scoped total. */
  accounts: readonly FilterOption[];
  memberIds: readonly string[];
  accountIds: readonly string[];
  onToggleMember: (id: string) => void;
  onToggleAccount: (id: string) => void;
  onClear: () => void;
}

/** The funnel in the header: who paid, and out of which shared account. */
export function FilterSheet({
  visible,
  onClose,
  members,
  accounts,
  memberIds,
  accountIds,
  onToggleMember,
  onToggleAccount,
  onClear,
}: FilterSheetProps) {
  const { t } = useI18n();
  return (
    <Sheet visible={visible} onClose={onClose} title={t("transactions.filter")}>
      <View className="gap-n5">
        <View className="gap-n3">
          <Kicker>{t("household.members")}</Kicker>
          <View className="flex-row flex-wrap gap-n2">
            {members.map((member) => (
              <Chip
                key={member.id}
                label={member.label}
                selected={memberIds.includes(member.id)}
                onPress={() => onToggleMember(member.id)}
              />
            ))}
          </View>
        </View>

        <View className="gap-n3">
          <Kicker>{t("accounts.sharedSection")}</Kicker>
          <View className="flex-row flex-wrap gap-n2">
            {accounts.map((account) => (
              <Chip
                key={account.id}
                label={account.label}
                selected={accountIds.includes(account.id)}
                onPress={() => onToggleAccount(account.id)}
              />
            ))}
          </View>
        </View>

        <View className="flex-row gap-n3">
          <Button title={t("common.all")} variant="ghost" onPress={onClear} className="flex-1" />
          <Button title={t("common.done")} onPress={onClose} className="flex-1" />
        </View>
      </View>
    </Sheet>
  );
}
