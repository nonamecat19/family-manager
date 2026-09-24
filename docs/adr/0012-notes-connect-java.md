# ADR 0012 — `services/notes` serves `notes.v1` over Connect from Spring

- Status: accepted
- Date: 2026-09-25

## Context

`libs/proto/notes/v1` was an **orphan contract**: `apps/notes` and the notes Telegram bot both
called it, the repo generated Go and TypeScript clients for it, and nothing served it. The graph
said so plainly — no `IMPLEMENTS` edge pointed at it — but the graph could not see the service
either, because it discovered services only from `go.work` members and `services/notes` is a
Spring app with no `go.mod`.

What `services/notes` actually exposed was a REST API (`/api/notes`, `/api/groups`) over a
different model: `Long` ids, a plain `content` string, `status` and `priority` enums, no blocks,
no stars, no archive. So the bot's `Блокноти` button answered "Не вийшло" — the Java service
returned 404 for `notes.v1.NotesService/ListNotebooks`, and would have for every other procedure.

## Decision

The Spring service implements `notes.v1` itself, alongside its existing REST controllers.

**Transport.** A single `ConnectController` maps `POST /{package}.{version}.{Service}/{Method}`
onto registered procedures. It accepts **both** Connect wire formats — binary protobuf, which the
Go clients send by default, and proto3 JSON, which the web and Expo clients send — and answers in
the format it was asked in. Errors are always JSON with a Connect `code`, as the protocol requires.
Services register procedures by extending `ConnectService`; the controller does not know about
notes specifically.

**Codegen.** `buf` now also generates Java (`buf.gen.java.yaml` → `sdk/java/`, package prefix
`com.nnc.familymanager`), driven from `just proto` like the Go and TypeScript SDKs. Gradle adds
that tree as a source directory through a `generatedProtoDir` property, so the Docker build can
point at its own copy. Generated code stays unedited, and the protobuf runtime version is pinned
to match what the plugin emits.

**Model.** Groups are notebooks. Blocks are stored as proto3 JSON in a new `notes.blocks` column
(migration `V3`), with `starred`, `archived` and `owner_user_id` beside it; `content` is still
written as flattened plain text so the REST API keeps working. A note written before blocks
existed decodes as one paragraph per line, so no backfill is needed. `preview`, `task_total` and
`task_done` are derived on read rather than stored.

**Scope.** Notebooks and notes CRUD, move, star, archive and search are implemented. Sharing,
comments, activity and image upload answer `unimplemented` explicitly rather than pretending —
the Java schema has nothing behind them, and `apps/notes` will need them before they are built.

**Transactions.** Procedures are dispatched through method references, which never pass through
Spring's `@Transactional` proxy, so each handler is wrapped in a `TransactionTemplate` at
registration. This is not a detail: with the annotation alone, mutations looked correct in the
response and were silently never committed.

## Alternatives

- **Point the bot at the REST API.** An afternoon's work, and the bot would have worked tonight.
  It leaves `notes.v1` orphaned, `apps/notes` broken, and the repo with two note models.
- **Rewrite notes as a Go service.** Matches the documented architecture and every other service,
  and would make the contract easy to serve completely — at the cost of discarding the Spring
  service, its tests and its REST consumers.
- **Shrink `notes.v1` to the Java model.** Honest about today's capability, but a breaking
  contract change that would force rework in `apps/notes` screens built against the richer model.
- **gRPC via a Spring gRPC starter.** Standard, but the apps and bots speak Connect over HTTP on
  `:8080`; a second protocol on a second port helps nobody here.

## Consequences

- `tools/repo-graph/extract.mjs` now discovers JVM services (`services/*/build.gradle`), so
  `service:notes` exists in the graph, owns its Flyway migrations, and `notes.v1` is no longer
  orphaned. No ontology change — a JVM service is still a `service`.
- The service has two public surfaces. The REST API is unchanged and still tested; new work
  belongs on the contract.
- Ids are database `Long`s rendered as decimal strings. Every other service uses UUIDs, so a
  future migration to UUID note ids would be a breaking change for stored client state.
- Search is `LIKE` over title and flattened text. It is honest for a family's notes and will not
  survive a large corpus.
- **Sharing.** `ShareNote` and `ShareNotebook` accept any well-formed UUID as `member_user_id`;
  the service has no family roster to check it against. A member share to a user outside the
  family is stored but inert: visibility also requires the note's `family_id` to equal the
  caller's token family, so the grant takes effect only if that user joins the family. Note
  images are public-read objects per ADR 0007, so their URLs stay readable to anyone who has
  them; `Unshare` revokes access to the note, not to images already seen. Both are accepted
  as-is for now, not oversights.
- Connect request bodies are capped at 12 MB (an 8 MB image as base64 JSON, plus headroom).
  The cap is checked against `Content-Length` before reading and enforced by a bounded read for
  chunked bodies; over it the call answers `resource_exhausted`.
