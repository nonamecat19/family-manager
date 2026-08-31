import { fromWire, type Account, type HiddenPrivateSummary } from "@fm/api";
import { View } from "react-native";

import { useI18n } from "@/components/i18n";
import { IconCircle, MoneyText, Row, iconOr, nocturne, tintFor, type IconName } from "@/components/nocturne";

/**
 * Screen 08's account card. The design draws every account as its own rounded surface with a
 * hairline outline — not a joined list — and gives the private ones a darker, quieter card, so
 * "this is not part of the family total" is legible before the caption is read.
 */

/** The glyph an account falls back to when it carries no stored icon. Kind, not name: a family
 * renaming "Mono" to "картка" must not change the picture. */
const KIND_ICON: Record<number, IconName> = {
  0: "wallet", // UNSPECIFIED
  1: "money", // CASH
  2: "credit-card", // CARD
  3: "credit-card", // BANK
  4: "piggy-bank", // SAVINGS
  5: "currency-btc", // CRYPTO
  6: "receipt", // DEBT
};

/** The private card's palette. Both values are Nocturne tokens, so the muted pass stays inside
 * the system rather than becoming a second set of greys. */
const PRIVATE_TINT = { bg: nocturne.neutral[800], fg: nocturne.neutral[400] } as const;
const HIDDEN_TINT = { bg: nocturne.neutral[900], fg: nocturne.neutral[600] } as const;

export interface AccountRowProps {
  account: Account;
  /** `private` is the caller's OWN private account: darker card plus the excluded caption. */
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
        backgroundColor: isPrivate ? nocturne.bg : nocturne.surface,
        borderWidth: 1,
        borderColor: isPrivate ? nocturne.neutral[900] : nocturne.neutral[800],
      }}
    >
      <Row
        title={account.name}
        // The design captions every private account with why its money is missing from the
        // headline. It is the whole point of the section, so it is not left to the kicker.
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
            // A negative balance is a debt, and the design draws it in the overspend hue —
            // the only place on this screen that colour is allowed.
            tone={balance.amountMinor < 0 ? "overspend" : "default"}
          />
        }
      />
    </View>
  );
}

/**
 * Another member's private accounts: a count and nothing else. The server never sends their
 * balances, and this row exists so the total is visibly incomplete rather than silently wrong.
 */
export function HiddenPrivateRow({ summary }: { summary: HiddenPrivateSummary }) {
  const { t } = useI18n();
  return (
    <View
      className="overflow-hidden rounded-lg"
      style={{
        backgroundColor: nocturne.bg,
        borderWidth: 1,
        borderColor: nocturne.neutral[900],
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
