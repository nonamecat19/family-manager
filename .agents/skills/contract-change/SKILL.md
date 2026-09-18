---
name: contract-change
description: Change a .proto contract safely — blast radius first, then regenerate Go stubs and the TypeScript SDK, then walk every dependent (service handlers, packages/api, apps), with a human gate on breaking changes. Use when editing anything under libs/proto or a service's proto/, adding or changing an rpc/message/field, or when the user asks to "add an endpoint" between an app and a service.
---

# contract-change

A contract change is **sequential work**: each step reads the previous step's generated output.
One agent, start to finish. Do not fan out.

## 0. Classify

| change | breaking? | gate |
|---|---|---|
| add a message, add an rpc | no | no |
| add a field with a new number | no | no |
| rename a field, change its type, reuse a number | **yes** | human gate |
| remove a field or rpc | **yes** | human gate |
| move a contract between files/packages | **yes** | human gate |

Field numbers are permanent. Removing a field means `reserved <n>;`, never reuse.

## 1. Blast radius BEFORE editing

```sh
just graph
just impact proto:libs/proto/<domain>/v1/<file>.proto
just impact 'rpc:<Service>.<Method>'
```

Write the dependent list down. It is the checklist for step 4 and the acceptance check at the
end — every node on it is either updated or explicitly deferred with a reason.

## 2. Edit the contract

Contracts live in `libs/proto/<domain>/v1/`. Required in every file:

```proto
syntax = "proto3";
package <domain>.v1;
// no go_package — buf managed mode sets it (libs/proto/buf.gen.yaml)
```

The directory is what binds a contract to its service in the graph: `libs/proto/<domain>/` →
`service:<domain>`. Keep the domain name and the service directory name identical, or ownership
falls back to inference at confidence 0.6 and `just impact` gets weaker.

Domain events count as contracts: payloads go in `libs/proto/<domain>/v1/events.proto`, subjects
are `<domain>.<entity>.<verb>`. Renaming a subject is breaking — consumers have durable
JetStream bindings.

## 3. Regenerate — never hand-edit generated code

```sh
just proto        # cd libs/proto && buf generate
just proto-check  # buf lint + buf breaking against master
```

Outputs (see docs/adr/0001-connectrpc.md): `sdk/go/**/*.pb.go` (protoc-gen-go),
`sdk/go/**/<d>v1connect/*.connect.go` (protoc-gen-connect-go), `sdk/typescript/src/**` 
(protoc-gen-es v2 — no separate client plugin; `createClient` builds the client from the
generated service descriptor).

Pre-migration services with a local `protoc` Makefile: `cd <service> && make proto`.
`sdk/*`, `*.pb.go` and `services/*/db/` are generated — edits there are lost on the next run.

## 4. Walk the dependents, in this order

1. **Implementing service** — `internal/grpc/server.go` (the Connect handler) maps proto types
   to handler types; handlers get the logic. Handlers never import proto types.
2. **`packages/api`** — wrap the new call in a TanStack Query hook over `createClient`, add
   query keys/invalidation. Apps never call the SDK directly.
3. **Apps** — consume via `packages/api`.
4. Tests at each layer.

## 5. Verify

```sh
just verify                    # go + ts + graph
just impact proto:<file>       # re-check: every node addressed?
git diff --stat                # generated churn should match the contract delta, nothing more
```

New rpcs must show up as `rpc:` nodes with `DECLARES` and `OWNED_BY` edges. If they do not,
`just proto` did not run or the file is outside the scanned paths.

## 6. Gate

Breaking change → stop and get explicit human approval before merging, with the blast-radius
list and the migration story for existing clients (mobile clients keep running old code — a
removed field breaks installed apps, not just this repo).
