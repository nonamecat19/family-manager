import { fromWire, type Account, type HiddenPrivateSummary } from "@fm/api";
import { View } from "react-native";

import { useI18n } from "@/components/i18n";
import { IconCircle, MoneyText, Row, tintFor } from "@/components/kit";
import { iconOr, organic, IconName } from "@fm/ui";

const KIND_ICON: Record<number, IconName> = {
  0: "wallet",
  1: "money",
  2: "credit-card",
  3: "credit-card",
  4: "piggy-bank",
  5: "currency-btc",
  6: "receipt",
};

const PRIVATE_TINT = { bg: organic.neutral[800], fg: organic.neutral[400] } as const;
const HIDDEN_TINT = { bg: organic.neutral[900], fg: organic.neutral[600] } as const;

export interface AccountRowProps {
  account: Account;
  tone?: "shared" | "private";
  onPress?: () => void;
}

export function AccountRow({ account, tone = "shared", onPress }: AccountRowProps) {
  const { t } = useI18n();
  const isPrivate = tone === "private";
  const balance = fromWire(account.balance, account.currencyCode);

  return (
    <View
      className="overflow-hidden rounded-lg"
      style={{
        backgroundColor: isPrivate ? organic.bg : organic.surface,
        borderWidth: 1,
        borderColor: isPrivate ? organic.neutral[900] : organic.neutral[800],
      }}
    >
      <Row
        title={account.name}
        subtitle={isPrivate ? t("accounts.excluded") : undefined}
        divider={false}
        onPress={onPress}
        leading={
          <IconCircle
            icon={iconOr(account.icon, KIND_ICON[account.kind] ?? "wallet")}
            size={34}
            tint={isPrivate ? PRIVATE_TINT : tintFor(account.colorStep)}
          />
        }
        trailing={
          <MoneyText
            value={balance}
            size={14}
            tone={balance.amountMinor < 0 ? "overspend" : "default"}
          />
        }
      />
    </View>
  );
}

export function HiddenPrivateRow({ summary }: { summary: HiddenPrivateSummary }) {
  const { t } = useI18n();
  return (
    <View
      className="overflow-hidden rounded-lg"
      style={{
        backgroundColor: organic.bg,
        borderWidth: 1,
        borderColor: organic.neutral[900],
        opacity: 0.75,
      }}
    >
      <Row
        title={t("accounts.othersPrivate", { name: summary.memberDisplayName })}
        subtitle={t("accounts.hiddenMeta", {
          accounts: t("common.accountCount", { count: summary.accountCount }),
        })}
        divider={false}
        leading={<IconCircle icon="lock-simple" size={34} tint={HIDDEN_TINT} />}
      />
    </View>
  );
}
