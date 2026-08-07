# ADR 0001 — ConnectRPC for app↔service, gRPC between services

- Status: accepted
- Date: 2026-08-06

## Context

Expo apps cannot speak native gRPC: React Native has no HTTP/2 trailer support, so grpc-web
needs a proxy (Envoy) and a browser shim. Services, on the other hand, benefit from real gRPC.
We want **one** contract definition for both.

## Decision

Protobuf in `libs/proto/<domain>/v1/` is the single contract surface. buf generates:

- Go server handlers with **protoc-gen-connect-go**, served on `:8080`. A Connect handler
  speaks Connect, gRPC and gRPC-Web on the same port.
- TypeScript clients with **protoc-gen-es v2**, used through `@connectrpc/connect` +
  `createConnectTransport` inside `packages/api`.
- Service-to-service calls use the gRPC protocol against the same handlers (`:9090` when a
  service wants a separate internal listener).

Apps therefore send ordinary HTTP/1.1 JSON requests, debuggable with curl, while services keep
binary gRPC.

## Alternatives

- **grpc-gateway REST** — adds `google.api.http` annotations to every rpc plus a second process,
  and the TS client is generated from OpenAPI rather than the proto, so the contract splits in
  two.
- **Plain REST + OpenAPI** — would drop protobuf entirely. Rejected because two hand-maintained
  sides drift, and the repo knowledge graph loses `DECLARES`/`USES_MESSAGE` edges that make
  `just impact` useful for endpoint changes.

## Consequences

- `just proto` is a prerequisite for building anything after a contract edit.
- `buf breaking` gates removed/renamed/renumbered fields; installed mobile clients keep running
  old code, so breaking changes need the human gate (see `.claude/skills/contract-change`).
- Anything not expressible in protobuf (file uploads) goes to MinIO with a presigned URL handed
  out by an rpc — not multipart through Connect.
