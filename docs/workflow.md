# Workflow — graph engineering applied to this repo

Two graphs run this repo. The **knowledge graph** says what exists and what depends on what.
The **task graph** says how a change moves from request to merge. Every rule below exists
because it removes a specific failure.

## 1. Knowledge graph: pipeline stages

The 9-stage pipeline, instantiated here (`tools/repo-graph/extract.mjs` implements 4-8):

| stage | here |
|---|---|
| 1 scope & value test | queries are multi-hop ("change this rpc → which apps rebuild") → graph earns its keep |
| 2 representation | property graph, JSON on disk (`docs/graph/graph.json`), no DB to run |
| 3 ontology | `docs/ontology.yaml` — 12 entity types, 10 relation types, precise verbs |
| 4 entity extraction | package.json / go.mod / .proto / .sql / docker-compose.yml |
| 5 relation extraction | workspace deps, direct go deps, rpc declarations, table reads/writes |
| 6 event extraction | not used — this domain is structural, not temporal (commits are in git) |
| 7 quality gate | domain/range validation drops bad edges into `graph.json.dropped`; sample 30 edges, open the cited `source:line` |
| 8 fusion | module path == directory == service name collapse to one node; duplicate edges collapse, first provenance wins |
| 9 serve | `just impact`, `graph.mmd` for humans, `graph.json` for agents |

Rules that are not negotiable:

- **Ontology before extraction.** Extracting first produces a word cloud with arrows.
- **Provenance on every fact.** `source`, `line`, `commit`, `extracted_at`. If an agent cannot
  cite the line, the edge does not exist.
- **Indirect dependencies stay out.** They are noise, not architecture.
- **Low-confidence edges are marked, not hidden** (`confidence < 1` on inferred ownership,
  env-file infra matches, and SQL reads).

### Reading the graph as an agent

```sh
just graph                     # rebuild before reasoning about structure
just impact proto:libs/proto/family/v1/family.proto
jq '.nodes[] | select(.type=="service")' docs/graph/graph.json
jq '.edges[] | select(.rel=="PERSISTS_TO")' docs/graph/graph.json
jq '.dropped' docs/graph/graph.json      # extractor's own confessions
```

Two health queries worth running periodically:

```sh
# contracts nobody implements (dead) — a node with no incoming IMPLEMENTS
jq -r '.nodes[]|select(.type=="contract")|.id' docs/graph/graph.json \
  | while read c; do grep -q "\"to\": \"$c\"" docs/graph/graph.json || echo "orphan: $c"; done
```

## 2. Task graph: the shape of a change

```
                     ┌──────────────────┐
request ─→ scope ─→  │ impact query     │ ─→ split decision
             │       └──────────────────┘         │
             │                                    ├─ independent? fan out
             │                                    └─ sequential?  one agent
             ▼                                    │
        acceptance                    ┌─ worker A ─┐
        criteria                      ├─ worker B ─┼─→ verify (fresh context)
                                      └─ worker C ─┘         │
                                                             ▼
                                                  merge (one owner) ─→ human gate ─→ commit
```

### The split decision

Ask: *where does this work split into pieces that never read each other's results?* Split only
that. Multi-agent teams beat a single agent by a wide margin on genuinely independent work and
lose on sequential work — every configuration, in controlled study. Uncoordinated agents amplify
each other's errors; one owner of the merge cuts that by ~4×.

| change | shape |
|---|---|
| contract change | **sequential** — proto → `just proto` → Go side → TS side. One agent. |
| new service | scaffold (1 agent) → then parallel: handlers / migrations / tests |
| new app screen using existing rpc | parallel with any backend work — the contract is fixed |
| dependency bump in `packages/config` | sequential + human gate |
| repo-wide rename | one agent, one writer per file, no exceptions |

### Fake edges

For every "and then" in a plan, ask whether the next job reads the previous job's output. Writing
service tests does not read the app's screen code — that edge is fake, delete it, run them in
parallel. Most hand-written plans carry two or three fake edges.

### The verify node

Non-negotiable and **in a separate context from the writer**. A model grading its own work in its
own context misses most of its own mistakes. Verification here is:

1. `just verify` — build, vet, test, lint, typecheck, graph rebuild.
2. A reviewer agent that did not write the diff, given a different question than the author's
   ("does the source line actually assert this?", "what breaks that the graph says depends on
   this?", "is the migration reversible?").
3. Diverse skeptics beat identical ones — give each verifier one question, not a checklist.

### The human gate

Route only irreversible edges through it:

- database migrations (destructive or not — ordering is irreversible in prod)
- `.proto` changes that remove/rename/renumber fields
- publishing SDK versions, deploys, anything in `infra/`
- deleting a node the graph shows as depended-upon

Not gated: ordinary code, tests, docs, additive proto fields, new packages. A gate on everything
makes the human the bottleneck; a gate on nothing means nobody is watching.

### Guardrails

1. Loops capped at 3 rounds; after that, escalate to the human with what remains failing.
2. One writer per file. Two jobs never mutate the same artifact.
3. Routing lives in written steps (this file); the model fills the jobs, not the plan.
4. Cap on spawned agents: 4 for a normal change, and never spawn to do sequential work.
5. Judge on numbers that cannot argue back — tests that ran, graph nodes that changed — never
   on an agent's self-report.

## 3. Per-change task graphs

### Contract change (sequential, one agent)

```
edit libs/proto/<d>/v1/*.proto
  → just impact proto:<file>        # list every dependent BEFORE regenerating
  → just proto                      # buf → sdk/go (connect handlers) + sdk/typescript
  → just proto-check                # buf lint + breaking-change detection
  → update implementing service (handlers, tests)
  → update packages/api wrapper + consuming apps
  → just verify
  → human gate if breaking
```

### New Go service

```
scaffold (skill: new-service)
  → register in go.work, docker-compose, justfile
  → just graph                      # the service must appear as a node
  ├─ migrations + sqlc              ┐
  ├─ grpc handlers                  ├─ parallel, disjoint files
  └─ config + wiring                ┘
  → verify (fresh context) → merge → human gate for the migration
```

### New Expo app

```
scaffold (skill: new-expo-app)
  → wire packages/{ui,auth,api,theme,config} as workspace:* deps
  → just graph                      # app node + DEPENDS_ON edges must appear
  ├─ screens/navigation             ┐
  └─ api integration via sdk        ┘ parallel only if the contract is unchanged
  → just check-ts → verify → merge
```

### Schema migration

```
write migration (additive first, always)
  → just impact table:<name>        # every service touching the table
  → sqlc generate for each affected service
  → backfill plan written down
  → HUMAN GATE
  → apply
```
