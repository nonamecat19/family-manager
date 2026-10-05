import { toDisplayError, useApproveDeviceLogin, useDenyDeviceLogin } from "@fm/api";
import { Code, ConnectError } from "@connectrpc/connect";
import { useRouter } from "expo-router";
import { ApproveDeviceForm, ScreenHeader, ScrollBody, Screen } from "@fm/ui";

import { useI18n } from "../../components/i18n/index.tsx";

export default function ApproveDeviceScreen() {
  const router = useRouter();
  const { t } = useI18n();
  const approve = useApproveDeviceLogin();
  const deny = useDenyDeviceLogin();

  const approveDeviceError = (error: unknown): string => {
    const errCode = error instanceof ConnectError ? error.code : undefined;
    if (errCode === Code.NotFound) return t("approveDevice.notFound");
    if (errCode === Code.ResourceExhausted) return t("approveDevice.tooManyAttempts");
    return toDisplayError(error, t("approveDevice.failed")).message;
  };

  const busy = approve.isPending || deny.isPending;
  const error = approve.isError
    ? approveDeviceError(approve.error)
    : deny.isError
      ? approveDeviceError(deny.error)
      : null;

  return (
    <Screen>
      <ScrollBody>
        <ScreenHeader title={t("approveDevice.title")} onBack={() => router.back()} backLabel={t("common.back")} />

        <ApproveDeviceForm
          strings={{
            body: t("approveDevice.body"),
            placeholder: t("approveDevice.placeholder"),
            approve: t("approveDevice.approve"),
            deny: t("approveDevice.deny"),
            approved: t("approveDevice.approved"),
            denied: t("approveDevice.denied"),
          }}
          busy={busy}
          error={error}
          onApprove={(code, done) => approve.mutate(code, { onSuccess: done })}
          onDeny={(code, done) => deny.mutate(code, { onSuccess: done })}
        />
      </ScrollBody>
    </Screen>
  );
}
