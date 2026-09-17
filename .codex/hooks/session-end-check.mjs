#!/usr/bin/env node
// Stop hook: a session must not end with a stale graph or a half-applied migration.
// Blocks once (exit 2) with instructions; never loops — stop_hook_active short-circuits.

import { execFileSync } from "node:child_process";
import { readFileSync } from "node:fs";
import path from "node:path";

let input = {};
try { input = JSON.parse(readFileSync(0, "utf8")); } catch { process.exit(0); }
if (input.stop_hook_active) process.exit(0); // already re-prompted once — let it end

const root = process.env.CLAUDE_PROJECT_DIR ?? process.cwd();
const run = (cmd, args) => {
  try { return { ok: true, out: execFileSync(cmd, args, { cwd: root, encoding: "utf8", timeout: 30000 }) }; }
  catch (e) { return { ok: false, out: `${e.stdout ?? ""}${e.stderr ?? ""}` }; }
};

const problems = [];

const graph = run("node", [path.join(root, "tools/repo-graph/extract.mjs"), "--check"]);
if (!graph.ok) problems.push("docs/graph/graph.json is stale — run `just graph` and commit it.");

const RETIRED = /^(notes-service|.*-android)\//;
const status = run("git", ["status", "--porcelain"]);
if (status.ok) {
  const files = status.out
    .split("\n")
    .filter(Boolean)
    .filter((l) => !/^ ?D/.test(l))              // deletions are not pending work
    .map((l) => l.slice(3).replace(/^.* -> /, "")) // renames: keep the destination
    .map((f) => f.replace(/^"|"$/g, ""))
    .filter((f) => !RETIRED.test(f));            // dirs being retired, not maintained

  const migrations = files.filter((f) => /^services\/[^/]+\/internal\/db\/migrations\//.test(f));
  const protos = files.filter((f) => f.startsWith("libs/proto/") && f.endsWith(".proto"));
  const list = (a) => (a.length > 3 ? `${a.slice(0, 3).join(", ")} (+${a.length - 3} more)` : a.join(", "));

  if (migrations.length) problems.push(`uncommitted migration(s): ${list(migrations)} — human gate before these are applied.`);
  if (protos.length) problems.push(`contract changed (${list(protos)}) — confirm \`just proto\` ran and every node from \`just impact\` was handled.`);
}

if (!problems.length) process.exit(0);
console.error("Before finishing:\n" + problems.map((p) => `  - ${p}`).join("\n"));
process.exit(2);
