# Autonomy policy

The human writes a brief and reads escalations. Everything between is machine work. This file is
the contract that makes that safe: what runs unattended, what stops, and what happens when
something goes wrong at 3am with nobody watching.

**The policy is enforced by a script, not by good intentions**: `just gate-check` inspects the
staged diff and exits non-zero on anything in the STOP column. `/autopilot` runs it before every
commit and before every merge.

## Gate table

| verdict | change |
|---|---|
| **AUTO** | new service, package, app, screen, handler, test, doc |
| **AUTO** | additive contract change: new rpc, new message, new field with a fresh number |
| **AUTO** | additive migration (new table, nullable column) applied to the **local dev DB only** |
| **AUTO** | dependency added inside one service or one app |
| **STOP** | destructive SQL: `DROP`, `TRUNCATE`, `ALTER ... DROP`, `RENAME`, `NOT NULL` on an existing column |
| **STOP** | breaking contract change: removed, renamed or renumbered field or rpc |
| **STOP** | any edit under `services/auth/**` or `libs/go/auth/**` — auth is hand-reviewed, always |
| **STOP** | any `.env*`, credential, key, or token value |
| **STOP** | `infra/**`, `.github/workflows/**`, `docker-compose.yml` — deploy and CI surface |
| **STOP** | dependency bump in `packages/config` or a shared `libs/go/*` — repo-wide blast radius |
| **STOP** | deleting a file whose graph node has inbound edges |
| **STOP** | applying anything to a non-local database |

A STOP does not halt the run. It writes an entry to `ESCALATIONS.md`, marks the unit `blocked`,
and **the loop continues with the next unblocked unit**. A blocked unit is not a failed run.

## Caps

| cap | value | why |
|---|---|---|
| attempts per unit | 3 | after three failed verify cycles the approach is wrong, not the code |
| consecutive unit failures | 2 → stop the line | a systemic break (broken toolchain, bad plan) should not burn the whole backlog |
| parallel implementers | 4, worktree-isolated | matches the task-graph guardrail |
| units per run | unbounded, but each unit is one commit | commits are the recovery points |
| loop rounds inside a unit | 3 | then escalate |

## Isolation

- Every brief gets a branch: `auto/<brief-id>-<slug>`. **Never commit to master directly.**
- Every unit worked in parallel gets its own git worktree, so one-writer-per-file is enforced by
  the filesystem rather than by discipline.
- Migrations run against the local compose DB only. The MCP postgres server is read-only
  (`--access-mode=restricted`) and points at localhost. Nothing in this workflow has production
  credentials, by construction.

## Merge

`/autopilot` merges its own branch when **all** hold:

1. every unit is `done` or `blocked` (and no blocked unit is a dependency of a done one),
2. `just verify` green locally,
3. CI green on the branch (`gh pr checks --watch`),
4. `just gate-check` clean across the whole branch diff, not just the last commit,
5. at least one `verifier` pass with `VERDICT: pass`.

Then: squash merge, delete branch. If any condition fails, the PR stays open and an escalation
is written. The human reviews master history after the fact — that is the accepted trade for
speed, and the reason the STOP column is wide.

## Stop-the-line

Halt the entire run, write an escalation, and do not merge when:

- two units fail in a row,
- `just gate-check` reports a STOP the plan did not anticipate (the decomposition was wrong),
- the knowledge graph loses edges the run did not intend to remove,
- a verifier returns `blocker` twice on the same unit,
- CI fails for a reason not reproducible locally.

## Human touchpoints (all asynchronous)

1. `/brief <idea>` — the creative step. Five minutes, produces `docs/briefs/NNNN-<slug>.md`.
2. `ESCALATIONS.md` — read when convenient, answer in the file, then `/autopilot resume`.
3. Post-hoc review of master history. Optional, and the safety net is CI plus the gate table.
