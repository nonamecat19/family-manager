import { View } from "react-native";

import { useI18n } from "@/components/i18n";
import { ScrollSheet } from "@/components/kit";
import { Button, Chip, Kicker } from "@fm/ui";

export interface FilterOption {
  id: string;
  label: string;
}

export interface FilterSheetProps {
  visible: boolean;
  onClose: () => void;
  members: readonly FilterOption[];
  accounts: readonly FilterOption[];
  memberIds: readonly string[];
  accountIds: readonly string[];
  groups: readonly FilterOption[];
  groupIds: readonly string[];
  onToggleMember: (id: string) => void;
  onToggleAccount: (id: string) => void;
  onToggleGroup: (id: string) => void;
  onClear: () => void;
}

export function FilterSheet({
  visible,
  onClose,
  members,
  accounts,
  memberIds,
  accountIds,
  groups,
  groupIds,
  onToggleMember,
  onToggleAccount,
  onToggleGroup,
  onClear,
}: FilterSheetProps) {
  const { t } = useI18n();
  return (
    <ScrollSheet visible={visible} onClose={onClose} title={t("transactions.filter")}>
      <View className="gap-[16px]">
        <View className="gap-[8px]">
          <Kicker>{t("household.members")}</Kicker>
          <View className="flex-row flex-wrap gap-[8px]">
            {members.map((member) => (
              <Chip
                key={member.id}
                label={member.label}
                active={memberIds.includes(member.id)}
                onPress={() => onToggleMember(member.id)}
              />
            ))}
          </View>
        </View>

        <View className="gap-[8px]">
          <Kicker>{t("accounts.sharedSection")}</Kicker>
          <View className="flex-row flex-wrap gap-[8px]">
            {accounts.map((account) => (
              <Chip
                key={account.id}
                label={account.label}
                active={accountIds.includes(account.id)}
                onPress={() => onToggleAccount(account.id)}
              />
            ))}
          </View>
        </View>

        {groups.length > 0 ? (
          <View className="gap-[8px]">
            <Kicker>{t("transactions.groups")}</Kicker>
            <View className="flex-row flex-wrap gap-[8px]">
              {groups.map((group) => (
                <Chip
                  key={group.id}
                  label={group.label}
                  active={groupIds.includes(group.id)}
                  onPress={() => onToggleGroup(group.id)}
                />
              ))}
            </View>
          </View>
        ) : null}

        <View className="flex-row gap-[8px]">
          <Button title={t("common.all")} tone="quiet" onPress={onClear} />
          <View className="flex-1"><Button title={t("common.done")} onPress={onClose} /></View>
        </View>
      </View>
    </ScrollSheet>
  );
}
