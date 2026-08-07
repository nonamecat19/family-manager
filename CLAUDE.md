See [AGENTS.md](AGENTS.md) — repo map, the knowledge-graph and task-graph workflow, conventions,
commands, and the project skills. Read it before editing anything.

Two rules that are never skipped:

1. `just graph` then `just impact <node>` before editing under `services/`, `libs/`,
   `packages/`, `apps/`, or `libs/proto` — state the blast radius in your plan.
2. `just verify` before claiming done, and a reviewer in a fresh context for anything crossing
   a service/app/contract boundary.
