---
name: plan-graph
description: Turn a request into an explicit task DAG before writing code — scope, blast radius from the knowledge graph, split decision (what runs parallel vs stays sequential), verify node, merge owner, human gates. Use at the start of any change touching more than one file or crossing an app/service/contract boundary in this monorepo.
---

# plan-graph

Produce the plan as a graph, then execute it. Skipping to code on a multi-boundary change is
how contracts and consumers drift.

## Step 1 — scope with real node ids

```sh
just graph
just impact <node>     # for every node you intend to touch
```

Write down: nodes to touch, blast radius per node, one acceptance check per node
(a command that passes/fails, not a feeling).

## Step 2 — the split decision

Ask: **where does this work split into pieces that never read each other's results?** Split only
that. Everything sequential stays with one agent — parallelism on sequential work degrades it,
consistently and measurably.

Sequential (one agent, no fan-out):
- contract change → codegen → consumers
- schema migration → sqlc → handlers
- repo-wide rename or move
- anything where step N reads step N-1's generated output

Parallel (fan out, cap 4):
- a Go service and an Expo screen behind an **unchanged** contract
- independent packages with disjoint files
- tests for component A and component B
- docs alongside code

Then delete fake edges: for every "and then" in your draft plan, check whether the next job
actually reads the previous job's output. If not, the edge is fake — run them in parallel.

## Step 3 — draw it

```
scope ──┬─ worker A (files: ...) ─┐
        ├─ worker B (files: ...) ─┼─→ verify ─→ merge ─→ gate? ─→ done
        └─ worker C (files: ...) ─┘
```

One writer per file — if two workers list the same file, the split is wrong; merge them.

## Step 4 — verify node (never skipped)

- `just verify` (go build/vet/test, turbo lint/typecheck/test, graph rebuild)
- a reviewer in a **fresh context** that did not write the diff
- one sharp question per verifier, not a checklist: "does the cited source line actually assert
  this?" / "what does the graph say depends on this that the diff ignores?" / "is this
  migration reversible?"

## Step 5 — merge and gate

One owner merges. Route through the human only for irreversible edges: migrations, breaking
proto changes, deploys/infra, deleting a depended-upon node, dependency bumps in
`packages/config`. Everything else proceeds without a gate.

## Guardrails

- Loops capped at 3 rounds, then escalate with what still fails.
- Max 4 spawned agents; zero for sequential work.
- The routing lives in `docs/workflow.md`; fill the jobs, do not rewrite the plan mid-flight.
- Judge on commands that ran and graph deltas, never on self-reports.

## Output format

```
NODES        service:auth, proto:libs/proto/auth/v1/auth.proto
BLAST        app:family-manager, pkg:@fm/api  (from just impact)
SEQUENTIAL   proto edit → just proto → service handlers → pkg/api wrapper
PARALLEL     [service tests] [app screen] — disjoint files
VERIFY       just verify + reviewer(fresh): "check the impact list is fully addressed"
GATE         yes — removes a proto field (breaking)
ACCEPT       just check-go && just check-ts && just impact proto:... shows no unhandled node
```
