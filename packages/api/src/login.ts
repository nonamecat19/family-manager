import { timestampMs } from "@bufbuild/protobuf/wkt";
import { useMutation } from "@tanstack/react-query";
import { useCallback, useEffect, useRef, useState } from "react";

import { tokensFromResponse, useAuth, type Tokens } from "@fm/auth";
import {
  DeviceLoginKind,
  DeviceLoginStatus,
  type StartDeviceLoginResponse,
} from "@fm/sdk/auth/v1/auth_pb";

import {
  hasExpired,
  isTerminalPhase,
  pollIntervalSeconds,
  type DeviceLoginPhase,
} from "./deviceLogin.ts";
import { useClients } from "./provider.tsx";
import { normalizeBotUsername, telegramStartUrl } from "./telegram.ts";

export { DeviceLoginKind, DeviceLoginStatus, type StartDeviceLoginResponse };
export {
  DEFAULT_POLL_INTERVAL_S,
  SLOW_DOWN_STEP_S,
  isTerminalPhase,
  type DeviceLoginPhase,
} from "./deviceLogin.ts";

export function useStartDeviceLogin() {
  const { auth } = useClients();
  return useMutation({
    mutationFn: (kind: DeviceLoginKind = DeviceLoginKind.DEVICE) =>
      auth.startDeviceLogin({ kind }),
  });
}

export function useApproveDeviceLogin() {
  const { auth } = useClients();
  return useMutation({
    mutationFn: async (userCode: string) => (await auth.approveDeviceLogin({ userCode })).kind,
  });
}

export function useDenyDeviceLogin() {
  const { auth } = useClients();
  return useMutation({
    mutationFn: async (userCode: string) => {
      await auth.denyDeviceLogin({ userCode });
    },
  });
}

export interface DeviceLoginPollOptions {
  onApproved?: (tokens: Tokens) => void | Promise<void>;
}

export interface DeviceLoginPoll {
  phase: DeviceLoginPhase;
  tokens: Tokens | null;
  error: unknown;
}

const IDLE: DeviceLoginPoll = { phase: "idle", tokens: null, error: null };

export function usePollDeviceLogin(
  grant: StartDeviceLoginResponse | null | undefined,
  opts: DeviceLoginPollOptions = {},
): DeviceLoginPoll {
  const { auth } = useClients();
  const [state, setState] = useState<DeviceLoginPoll>(IDLE);
  const onApproved = useRef(opts.onApproved);

  useEffect(() => {
    onApproved.current = opts.onApproved;
  }, [opts.onApproved]);

  useEffect(() => {
    if (!grant?.deviceCode) {
      setState(IDLE);
      return;
    }

    let cancelled = false;
    let timer: ReturnType<typeof setTimeout> | undefined;
    let interval = pollIntervalSeconds(0, {
      slowDown: false,
      intervalSeconds: grant.intervalSeconds,
    });
    const expiresAt = grant.expiresAt ? timestampMs(grant.expiresAt) : null;
    let lastError: unknown = null;

    const settle = (next: DeviceLoginPoll) => {
      if (!cancelled) setState(next);
    };

    const tick = async () => {
      if (cancelled) return;
      if (hasExpired(expiresAt, Date.now())) {
        settle({ phase: "expired", tokens: null, error: lastError });
        return;
      }

      try {
        const res = await auth.pollDeviceLogin({ deviceCode: grant.deviceCode });
        if (cancelled) return;
        lastError = null;

        switch (res.status) {
          case DeviceLoginStatus.APPROVED: {
            const tokens = tokensFromResponse(res, Date.now());
            try {
              await onApproved.current?.(tokens);
              settle({ phase: "approved", tokens, error: null });
            } catch (error) {
              settle({ phase: "error", tokens, error });
            }
            return;
          }
          case DeviceLoginStatus.DENIED:
            settle({ phase: "denied", tokens: null, error: null });
            return;
          case DeviceLoginStatus.EXPIRED:
            settle({ phase: "expired", tokens: null, error: null });
            return;
          default:
            interval = pollIntervalSeconds(interval, {
              slowDown: res.status === DeviceLoginStatus.SLOW_DOWN,
              intervalSeconds: res.intervalSeconds,
            });
            settle({ phase: "pending", tokens: null, error: null });
        }
      } catch (error) {
        if (cancelled) return;
        lastError = error;
        settle({ phase: "pending", tokens: null, error });
      }

      timer = setTimeout(() => void tick(), interval * 1000);
    };

    setState({ phase: "pending", tokens: null, error: null });
    timer = setTimeout(() => void tick(), interval * 1000);

    return () => {
      cancelled = true;
      if (timer !== undefined) clearTimeout(timer);
    };
  }, [auth, grant]);

  return state;
}

export interface TelegramLoginOptions {
  bot: string | null | undefined;
  open: (url: string) => Promise<unknown>;
}

export interface TelegramLogin {
  available: boolean;
  phase: DeviceLoginPhase;
  userCode: string | null;
  error: unknown;
  start: () => Promise<void>;
  cancel: () => void;
}

export function useTelegramLogin({ bot, open }: TelegramLoginOptions): TelegramLogin {
  const name = normalizeBotUsername(bot);
  const { signIn } = useAuth();
  const startLogin = useStartDeviceLogin();
  const { mutateAsync: startGrant, reset: resetStart } = startLogin;
  const [grant, setGrant] = useState<StartDeviceLoginResponse | null>(null);
  const [openError, setOpenError] = useState<unknown>(null);

  const poll = usePollDeviceLogin(grant, { onApproved: signIn });

  const start = useCallback(async () => {
    if (!name) return;
    setGrant(null);
    setOpenError(null);
    try {
      const next = await startGrant(DeviceLoginKind.TELEGRAM);
      setGrant(next);
      await open(telegramStartUrl(name, next.telegramStartPayload));
    } catch (error) {
      setOpenError(error);
      setGrant(null);
    }
  }, [name, open, startGrant]);

  const cancel = useCallback(() => {
    setGrant(null);
    setOpenError(null);
    resetStart();
  }, [resetStart]);

  let phase: DeviceLoginPhase = poll.phase;
  if (startLogin.isPending) phase = "starting";
  else if (openError) phase = "error";

  return {
    available: name !== null,
    phase,
    userCode: grant && !isTerminalPhase(phase) ? grant.userCode : null,
    error: openError ?? startLogin.error ?? poll.error ?? null,
    start,
    cancel,
  };
}
