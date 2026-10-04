import { toDisplayError, useApproveDeviceLogin, useDenyDeviceLogin } from "@fm/api";
import { Code, ConnectError } from "@connectrpc/connect";
import { useRouter } from "expo-router";
import { ApproveDeviceForm, Screen, ScreenHeader, ScrollBody } from "@fm/ui";

import { strings } from "../../components/i18n/index.ts";

function approveDeviceError(error: unknown): string {
  const code = error instanceof ConnectError ? error.code : undefined;
  if (code === Code.NotFound) return strings.approveDevice.notFound;
  if (code === Code.ResourceExhausted) return strings.approveDevice.tooManyAttempts;
  return toDisplayError(error, strings.approveDevice.failed).message;
}

export default function ApproveDeviceScreen() {
  const router = useRouter();
  const approve = useApproveDeviceLogin();
  const deny = useDenyDeviceLogin();

  const busy = approve.isPending || deny.isPending;
  const error = approve.isError
    ? approveDeviceError(approve.error)
    : deny.isError
      ? approveDeviceError(deny.error)
      : null;

  return (
    <Screen>
      <ScrollBody>
        <ScreenHeader
          title={strings.approveDevice.title}
          onBack={() => router.back()}
          backLabel={strings.approveDevice.back}
        />

        <ApproveDeviceForm
          strings={{
            body: strings.approveDevice.body,
            placeholder: strings.approveDevice.placeholder,
            approve: strings.approveDevice.approve,
            deny: strings.approveDevice.deny,
            approved: strings.approveDevice.approved,
            denied: strings.approveDevice.denied,
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
