import { toDisplayError } from "@fm/api";
import { GateMessage } from "@fm/ui";

import { useI18n } from "../i18n/index.tsx";

export function LoadError({ error, onRetry }: { error: unknown; onRetry: () => void }) {
  const { t } = useI18n();
  const shown = toDisplayError(error, t("common.loadFailed"));
  return (
    <GateMessage
      body={shown.message}
      reference={shown.reference ? t("common.errorReference", { ref: shown.reference }) : undefined}
      actionTitle={t("common.tryAgain")}
      onAction={onRetry}
    />
  );
}
