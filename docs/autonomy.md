# Autonomy policy

The human writes a brief and reads escalations. Everything between is machine work. This file is
the contract that makes that safe: what runs unattended, what stops, and what happens when
something goes wrong at 3am with nobody watching.

**The policy is enforced by a script, not by good intentions**: `just gate-check` inspects the
staged diff and exits non-zero on anything in the STOP column. `/autopilot` runs it before every
commit and once over the whole run before it reports.

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

- Runs commit straight to local `master` — no branches, no PRs. `backlog.mjs begin` records the
  run's base sha, so `<base>..HEAD` is exactly the run's commits and one unit is one commit.
- Every unit worked in parallel gets its own detached git worktree, so one-writer-per-file is
  enforced by the filesystem rather than by discipline; the coordinator cherry-picks each
  accepted unit onto master.
- A run never pushes. Pushing master triggers the deploy, so it stays a human step.
- Migrations run against the local compose DB only. The MCP postgres server is read-only
  (`--access-mode=restricted`) and points at localhost. Nothing in this workflow has production
  credentials, by construction.

## Completion

A run is complete when **all** hold:

1. every unit is `done` or `blocked` (and no blocked unit is a dependency of a done one),
2. `just verify` green locally,
3. `just gate-check-range <base>..HEAD` clean across the whole run, not just the last commit,
4. at least one `verifier` pass with `VERDICT: pass`.

Then the run reports and stops; the human reviews `<base>..HEAD` and pushes. CI runs on the
pushed master and gates the deploy. If any condition fails, the commits stay on local master,
an escalation names the condition and the range, and the human decides whether to fix forward
or revert.

## Stop-the-line

Halt the entire run and write an escalation when:

- two units fail in a row,
- `just gate-check` reports a STOP the plan did not anticipate (the decomposition was wrong),
- the knowledge graph loses edges the run did not intend to remove,
- a verifier returns `blocker` twice on the same unit,

## Human touchpoints (all asynchronous)

1. `/brief <idea>` — the creative step. Five minutes, produces `docs/briefs/NNNN-<slug>.md`.
2. `ESCALATIONS.md` — read when convenient, answer in the file, then `/autopilot resume`.
3. Review `<base>..HEAD` on local master and push. CI on master then gates the deploy.
