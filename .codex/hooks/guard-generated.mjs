#!/usr/bin/env node
// PreToolUse guard: refuse hand-edits to generated code.
// Generated output is reproduced by `just proto` / `just sqlc <svc>`; editing it is silently
// lost on the next codegen run, which is the single most common agent mistake in this repo.
// Exit 2 blocks the tool call and feeds stderr back to the model.

import { readFileSync } from "node:fs";

// buf writes the sources under sdk/*, never the module manifests that make them a Go module
// or a pnpm package. Those are hand-authored infrastructure and must stay editable.
const SDK_MANIFESTS =
  /(^|\/)sdk\/(go\/(go\.mod|go\.sum)|typescript\/(package\.json|tsconfig\.json|README\.md))$/;

const RULES = [
  { re: /(^|\/)sdk\/(go|typescript)\//, fix: "run `just proto` — edit libs/proto/<domain>/v1/*.proto instead" },
  { re: /\.pb\.go$|\.connect\.go$|_pb\.ts$/, fix: "run `just proto` — edit the .proto, not the stub" },
  { re: /(^|\/)services\/[^/]+\/db\//, fix: "run `just sqlc <service>` — edit internal/db/queries/*.sql instead" },
  { re: /(^|\/)docs\/graph\//, fix: "run `just graph` — this file is extracted, never authored" },
  { re: /(^|\/)\.env$/, fix: "edit .env.example; real .env files stay out of the repo" },
];

let input = {};
try { input = JSON.parse(readFileSync(0, "utf8")); } catch { process.exit(0); }

const p = input?.tool_input?.file_path ?? input?.tool_input?.path ?? "";
if (!p) process.exit(0);

if (SDK_MANIFESTS.test(p)) process.exit(0);

const hit = RULES.find((r) => r.re.test(p));
if (!hit) process.exit(0);

console.error(
  `BLOCKED: ${p} is generated and must not be hand-edited.\n` +
  `Fix: ${hit.fix}\n` +
  `See docs/stack.md and .claude/skills/contract-change.`,
);
process.exit(2);
