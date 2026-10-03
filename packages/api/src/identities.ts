import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";

import type { Identity } from "@fm/sdk/auth/v1/auth_pb";

import { useClients } from "./provider.tsx";
import { queryKeys } from "./queryKeys.ts";
import {
  LINK_POLL_MS,
  LINK_WAIT_MS,
  TELEGRAM_PROVIDER,
  normalizeBotUsername,
  telegramStartUrl,
} from "./telegram.ts";

export function useIdentities(opts: { enabled?: boolean; refetchInterval?: number | false } = {}) {
  const { auth } = useClients();
  return useQuery({
    queryKey: queryKeys.identities(),
    queryFn: async () => (await auth.listIdentities({})).identities,
    enabled: opts.enabled ?? true,
    refetchInterval: opts.refetchInterval ?? false,
  });
}

export function useCreateLinkToken() {
  const { auth } = useClients();
  return useMutation({
    mutationFn: async (provider: string) => (await auth.createLinkToken({ provider })).token,
  });
}

export function useUnlinkIdentity() {
  const { auth } = useClients();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { provider: string; externalId: string }) => auth.unlink(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.identities() }),
  });
}

export interface TelegramLinkOptions {
  bot: string | null | undefined;
  open: (url: string) => Promise<unknown>;
}

export interface TelegramLink {
  available: boolean;
  loading: boolean;
  identity: Identity | null;
  awaiting: boolean;
  connecting: boolean;
  unlinking: boolean;
  error: unknown;
  connect: () => Promise<void>;
  unlink: () => Promise<void>;
}

export function useTelegramLink({ bot, open }: TelegramLinkOptions): TelegramLink {
  const name = normalizeBotUsername(bot);
  const [awaitingSince, setAwaitingSince] = useState<number | null>(null);

  const identities = useIdentities({
    enabled: name !== null,
    refetchInterval: awaitingSince !== null ? LINK_POLL_MS : false,
  });
  const refetchIdentities = identities.refetch;
  const createToken = useCreateLinkToken();
  const unlinkIdentity = useUnlinkIdentity();

  const identity =
    identities.data?.find((candidate) => candidate.provider === TELEGRAM_PROVIDER) ?? null;

  useEffect(() => {
    if (awaitingSince === null) return;
    if (identity) {
      setAwaitingSince(null);
      return;
    }
    const left = awaitingSince + LINK_WAIT_MS - Date.now();
    const timer = setTimeout(() => {
      setAwaitingSince(null);
      void refetchIdentities();
    }, Math.max(left, 0));
    return () => clearTimeout(timer);
  }, [awaitingSince, identity, refetchIdentities]);

  const connect = async () => {
    if (!name) return;
    const token = await createToken.mutateAsync(TELEGRAM_PROVIDER);
    await open(telegramStartUrl(name, token));
    setAwaitingSince(Date.now());
  };

  const unlink = async () => {
    if (!identity) return;
    setAwaitingSince(null);
    await unlinkIdentity.mutateAsync({
      provider: identity.provider,
      externalId: identity.externalId,
    });
  };

  return {
    available: name !== null,
    loading: name !== null && identities.isPending,
    identity,
    awaiting: awaitingSince !== null,
    connecting: createToken.isPending,
    unlinking: unlinkIdentity.isPending,
    error: createToken.error ?? unlinkIdentity.error ?? identities.error ?? null,
    connect,
    unlink,
  };
}
