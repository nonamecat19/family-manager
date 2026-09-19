import { createClient, type Client, type Interceptor } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";

import { requestIdInterceptor } from "./requestId.ts";

import { AuthService } from "@fm/sdk/auth/v1/auth_pb";
import { FamilyService } from "@fm/sdk/family/v1/family_pb";
import { FinanceService } from "@fm/sdk/finance/v1/finance_pb";
import { NotesService } from "@fm/sdk/notes/v1/notes_pb";
import { NotificationsService } from "@fm/sdk/notifications/v1/notifications_pb";
import { RecipesService } from "@fm/sdk/recipes/v1/recipes_pb";
import { TasksService } from "@fm/sdk/tasks/v1/tasks_pb";

const PUBLIC_PROCEDURES = new Set([
  `${AuthService.typeName}/${AuthService.method.login.name}`,
  `${AuthService.typeName}/${AuthService.method.register.name}`,
  `${AuthService.typeName}/${AuthService.method.refresh.name}`,
  `${AuthService.typeName}/${AuthService.method.startDeviceLogin.name}`,
  `${AuthService.typeName}/${AuthService.method.pollDeviceLogin.name}`,
]);

export type ServiceName = "auth" | "family" | "finance" | "recipes" | "notes" | "notifications" | "tasks";

export interface ClientsOptions {
  baseUrl: string;
  serviceUrls?: Partial<Record<ServiceName, string>>;
  getAccessToken: () => Promise<string | null>;
}

function createTransport(url: string, getAccessToken: ClientsOptions["getAccessToken"]) {
  const auth: Interceptor = (next) => async (req) => {
    if (!PUBLIC_PROCEDURES.has(`${req.service.typeName}/${req.method.name}`)) {
      const token = await getAccessToken();
      if (token) req.header.set("Authorization", `Bearer ${token}`);
    }
    return next(req);
  };

  return createConnectTransport({
    baseUrl: url,
    useBinaryFormat: false,
    interceptors: [requestIdInterceptor, auth],
  });
}

export interface Clients {
  auth: Client<typeof AuthService>;
  family: Client<typeof FamilyService>;
  finance: Client<typeof FinanceService>;
  recipes: Client<typeof RecipesService>;
  notes: Client<typeof NotesService>;
  notifications: Client<typeof NotificationsService>;
  tasks: Client<typeof TasksService>;
}

export function createClients(opts: ClientsOptions): Clients {
  const urlFor = (service: ServiceName) =>
    opts.serviceUrls?.[service] ?? opts.baseUrl;

  return {
    auth: createClient(AuthService, createTransport(urlFor("auth"), opts.getAccessToken)),
    family: createClient(FamilyService, createTransport(urlFor("family"), opts.getAccessToken)),
    finance: createClient(FinanceService, createTransport(urlFor("finance"), opts.getAccessToken)),
    recipes: createClient(RecipesService, createTransport(urlFor("recipes"), opts.getAccessToken)),
    notes: createClient(NotesService, createTransport(urlFor("notes"), opts.getAccessToken)),
    notifications: createClient(
      NotificationsService,
      createTransport(urlFor("notifications"), opts.getAccessToken),
    ),
    tasks: createClient(TasksService, createTransport(urlFor("tasks"), opts.getAccessToken)),
  };
}
