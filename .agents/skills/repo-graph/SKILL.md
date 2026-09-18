---
name: repo-graph
description: Build and query the repo knowledge graph (docs/graph/graph.json) — blast radius before an edit, dependency/ownership questions, orphan contracts, staleness checks. Use whenever you need to know what depends on what in this monorepo, before editing anything under services/, libs/, packages/, apps/, or libs/proto, or when the user asks "what breaks if", "who uses", "who owns", "what does X depend on".
---

# repo-graph

Answer structural questions from the graph, never from guessing or a grep sweep.

## Rebuild first

```sh
just graph        # or: node tools/repo-graph/extract.mjs
```

Cheap (<1s, no deps). Always rebuild before reasoning — a stale graph is worse than none.

## Blast radius (the main use)

```sh
just impact service:auth
just impact table:users
just impact 'rpc:AuthService.Login'
just impact proto:libs/proto/family/v1/family.proto
```

Prints every node that transitively depends on the target, with the edge and the source file
that asserts it. **Quote this list in your plan before editing.** An unknown node id prints the
full candidate list — use it to find the right id rather than guessing.

## Direct queries

```sh
jq '.nodes[] | select(.type=="service") | .id' docs/graph/graph.json
jq '.edges[] | select(.from=="app:shopping")' docs/graph/graph.json
jq '.edges[] | select(.rel=="PERSISTS_TO" and .to=="table:users")' docs/graph/graph.json
jq '.nodes[] | select(.props.confidence != null)' docs/graph/graph.json   # inferred, verify
jq '.dropped' docs/graph/graph.json      # edges the ontology rejected
```

Node ids: `app:<dir>` `pkg:<name>` `service:<name>` `golib:<name>` `gomod:<path>`
`proto:<file>` `rpc:<Svc>.<Method>` `msg:<pkg>.<Name>` `table:<name>` `migration:<file>`
`infra:<name>` `ext:<name>`.

## Verify before you trust

Every node and edge carries `source`, `line`, `commit`. If a claim matters, open the cited
line. Edges with `confidence < 1` are inferred (contract ownership without `go_package`, infra
matched from an env file, SQL reads) — verify those before acting on them.

## Extending the graph

The extractor mirrors `docs/ontology.yaml`. To add a type or relation:

1. Edit `docs/ontology.yaml` (entity or relation, with domain/range and a precise verb name).
2. Mirror it in the `ENTITY_TYPES` / `RELATIONS` constants at the top of
   `tools/repo-graph/extract.mjs`.
3. Add the extraction pass (`extractNode`, `extractGo`, `extractProto`, `extractSql`,
   `extractInfra` are the models to copy). Emit `source` and `line` on every fact.
4. `just graph`, then check `jq '.dropped'` is empty — a non-empty drop list means the new
   edges violate domain/range.
5. Quality gate: sample 30 new edges, open the cited line, confirm the file literally asserts
   the edge. Below ~90% precision, fix the extractor, not the output.

Never hand-edit `docs/graph/*` — it is generated.

## CI

`node tools/repo-graph/extract.mjs --check` exits 1 when the committed graph is stale.
