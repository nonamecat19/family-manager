import { createClient, type Client, type Interceptor } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";

import { AuthService } from "@fm/sdk/auth/v1/auth_pb";
import { FamilyService } from "@fm/sdk/family/v1/family_pb";
import { FinanceService } from "@fm/sdk/finance/v1/finance_pb";

/**
 * Procedures that must go out without a token, or login could never happen. Keyed by
 * "<service>/<method>" — the same identity the Go interceptor's exemption list uses.
 */
const PUBLIC_PROCEDURES = new Set([
  `${AuthService.typeName}/${AuthService.method.login.name}`,
  `${AuthService.typeName}/${AuthService.method.register.name}`,
  `${AuthService.typeName}/${AuthService.method.refresh.name}`,
]);

export interface ClientsOptions {
  /** Gateway base URL, e.g. https://api.example.com. */
  baseUrl: string;
  /** Resolves a fresh access token, or null when anonymous. Supplied by @fm/auth. */
  getAccessToken: () => Promise<string | null>;
}

/**
 * The Connect transport used by every hook. JSON over HTTP/1.1 (see docs/adr/0001-connectrpc.md)
 * so responses stay readable in a proxy and React Native needs no HTTP/2 shims.
 */
function createTransport({ baseUrl, getAccessToken }: ClientsOptions) {
  const auth: Interceptor = (next) => async (req) => {
    if (!PUBLIC_PROCEDURES.has(`${req.service.typeName}/${req.method.name}`)) {
      const token = await getAccessToken();
      if (token) req.header.set("Authorization", `Bearer ${token}`);
    }
    return next(req);
  };

  return createConnectTransport({ baseUrl, useBinaryFormat: false, interceptors: [auth] });
}

export interface Clients {
  auth: Client<typeof AuthService>;
  family: Client<typeof FamilyService>;
  finance: Client<typeof FinanceService>;
}

/** Builds one client per service. Call once per app and put the result in a context. */
export function createClients(opts: ClientsOptions): Clients {
  const transport = createTransport(opts);
  return {
    auth: createClient(AuthService, transport),
    family: createClient(FamilyService, transport),
    finance: createClient(FinanceService, transport),
  };
}
