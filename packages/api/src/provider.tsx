import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createContext, useContext, useMemo, type ReactNode } from "react";

import { createClients, type Clients, type ClientsOptions } from "./client.ts";
import { shouldRetryQuery } from "./retry.ts";

const ClientsContext = createContext<Clients | null>(null);

export interface ApiProviderProps extends ClientsOptions {
  /** Supply your own QueryClient in tests; otherwise one is created here. */
  queryClient?: QueryClient;
  children: ReactNode;
}

/** Defaults tuned for a mobile ledger: data is small, connectivity is not guaranteed. */
export function createQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 30_000,
        gcTime: 24 * 60 * 60_000,
        // Not a count: `retry: 2` retried everything, including the codes that are answers.
        // "That email is already registered" took three round trips to reach the user.
        retry: shouldRetryQuery,
        refetchOnWindowFocus: false,
      },
      mutations: { retry: 0 },
    },
  });
}

export function ApiProvider({
  baseUrl,
  serviceUrls,
  getAccessToken,
  queryClient,
  children,
}: ApiProviderProps) {
  const clients = useMemo(
    () => createClients({ baseUrl, serviceUrls, getAccessToken }),
    [baseUrl, serviceUrls, getAccessToken],
  );
  const qc = useMemo(() => queryClient ?? createQueryClient(), [queryClient]);

  return (
    <QueryClientProvider client={qc}>
      <ClientsContext.Provider value={clients}>{children}</ClientsContext.Provider>
    </QueryClientProvider>
  );
}

export function useClients(): Clients {
  const ctx = useContext(ClientsContext);
  if (!ctx) throw new Error("useClients: no <ApiProvider> above this component");
  return ctx;
}
