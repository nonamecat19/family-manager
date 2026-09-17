#!/usr/bin/env node
// PostToolUse: re-extract the knowledge graph when a STRUCTURAL file changes.
// Narrow on purpose — editing a handler does not move a node, so it does not run.
// Keeps docs/graph/graph.json current so `just impact` and CI's graph job never disagree.

import { execFileSync } from "node:child_process";
import { readFileSync } from "node:fs";
import path from "node:path";

const STRUCTURAL = /(^|\/)(package\.json|go\.mod|go\.work|docker-compose(\.services)?\.yml|pnpm-workspace\.yaml)$|\.proto$|\.sql$/;

let input = {};
try { input = JSON.parse(readFileSync(0, "utf8")); } catch { process.exit(0); }

const p = input?.tool_input?.file_path ?? "";
if (!p || !STRUCTURAL.test(p)) process.exit(0);
if (/node_modules/.test(p)) process.exit(0);

const root = process.env.CLAUDE_PROJECT_DIR ?? process.cwd();
try {
  const out = execFileSync("node", [path.join(root, "tools/repo-graph/extract.mjs")], {
    cwd: root, encoding: "utf8", timeout: 30000,
  }).trim();
  console.log(`[graph] ${out}`);
} catch (e) {
  console.error(`[graph] extraction failed: ${e.message}`);
  process.exit(1); // non-blocking: surfaced to the user, the tool call still stands
}
