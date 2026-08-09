import { createClient, type Client, type Interceptor } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";

import { AuthService } from "@fm/sdk/auth/v1/auth_pb";
import { FamilyService } from "@fm/sdk/family/v1/family_pb";
import { FinanceService } from "@fm/sdk/finance/v1/finance_pb";
import { RecipesService } from "@fm/sdk/recipes/v1/recipes_pb";

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
  /**
   * Base URL for every service. Correct behind a gateway that routes by procedure path,
   * which is the deployed shape (Caddy — docs/adr/0004-compose-vps.md).
   */
  baseUrl: string;
  /**
   * Per-service overrides. In development each service listens on its own port and there is
   * no gateway, so without these every call would land on whichever service baseUrl names.
   */
  serviceUrls?: Partial<Record<"auth" | "family" | "finance" | "recipes", string>>;
  /** Resolves a fresh access token, or null when anonymous. Supplied by @fm/auth. */
  getAccessToken: () => Promise<string | null>;
}

/**
 * The Connect transport used by every hook. JSON over HTTP/1.1 (see docs/adr/0001-connectrpc.md)
 * so responses stay readable in a proxy and React Native needs no HTTP/2 shims.
 */
function createTransport(url: string, getAccessToken: ClientsOptions["getAccessToken"]) {
  const auth: Interceptor = (next) => async (req) => {
    if (!PUBLIC_PROCEDURES.has(`${req.service.typeName}/${req.method.name}`)) {
      const token = await getAccessToken();
      if (token) req.header.set("Authorization", `Bearer ${token}`);
    }
    return next(req);
  };

  return createConnectTransport({ baseUrl: url, useBinaryFormat: false, interceptors: [auth] });
}

export interface Clients {
  auth: Client<typeof AuthService>;
  family: Client<typeof FamilyService>;
  finance: Client<typeof FinanceService>;
  recipes: Client<typeof RecipesService>;
}

/** Builds one client per service. Call once per app and put the result in a context. */
export function createClients(opts: ClientsOptions): Clients {
  const urlFor = (service: "auth" | "family" | "finance" | "recipes") =>
    opts.serviceUrls?.[service] ?? opts.baseUrl;

  return {
    auth: createClient(AuthService, createTransport(urlFor("auth"), opts.getAccessToken)),
    family: createClient(FamilyService, createTransport(urlFor("family"), opts.getAccessToken)),
    finance: createClient(FinanceService, createTransport(urlFor("finance"), opts.getAccessToken)),
    recipes: createClient(RecipesService, createTransport(urlFor("recipes"), opts.getAccessToken)),
  };
}
