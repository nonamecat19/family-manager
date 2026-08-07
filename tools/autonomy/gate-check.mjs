#!/usr/bin/env node
// Deterministic enforcement of docs/autonomy.md. Inspects a diff and prints AUTO or STOP.
//
//   node tools/autonomy/gate-check.mjs              # staged changes (pre-commit)
//   node tools/autonomy/gate-check.mjs --range A..B # a whole branch (pre-merge)
//
// exit 0 = AUTO (unattended commit/merge allowed)
// exit 3 = STOP (write an escalation, block the unit, keep going with other units)
// exit 1 = the check itself failed — treat as STOP.

import { execFileSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";

const ROOT = path.resolve(import.meta.dirname, "../..");
const args = process.argv.slice(2);
const rangeIdx = args.indexOf("--range");
const RANGE = rangeIdx !== -1 ? args[rangeIdx + 1] : null;

const git = (...a) => execFileSync("git", a, { cwd: ROOT, encoding: "utf8", maxBuffer: 32 << 20 });
const diffArgs = RANGE ? [RANGE] : ["--cached"];

let files, patch;
try {
  files = git("diff", ...diffArgs, "--name-status").split("\n").filter(Boolean);
  // Explicit prefixes: a user's diff.mnemonicPrefix would otherwise emit c//i/ and break parsing.
  patch = git("diff", ...diffArgs, "-U0", "--src-prefix=a/", "--dst-prefix=b/");
} catch (e) {
  console.error(`gate-check: cannot read diff (${e.message})`);
  process.exit(1);
}

const stops = [];
const stop = (rule, detail) => stops.push({ rule, detail });

// --- path-based rules -------------------------------------------------------
const PATH_RULES = [
  [/^services\/auth\//, "auth service is hand-reviewed, always"],
  [/^libs\/go\/auth\//, "shared auth library is hand-reviewed, always"],
  [/(^|\/)\.env($|\.)(?!example)/, "environment/credential file"],
  [/^infra\//, "deploy surface"],
  [/^\.github\/workflows\//, "CI surface"],
  [/^docker-compose\.yml$/, "runtime topology"],
  [/^packages\/config\//, "repo-wide blast radius"],
  [/^(package\.json|pnpm-lock\.yaml|go\.work)$/, "workspace-wide dependency change"],
];

const deletions = [];
const changed = files.map((l) => {
  const [status, ...rest] = l.split("\t");
  return { status: status[0], file: rest[rest.length - 1] };
});

for (const { status, file } of changed) {
  for (const [re, why] of PATH_RULES) if (re.test(file)) stop(`path:${file}`, why);
  if (status === "D") deletions.push(file);
}

// --- deleted nodes with inbound edges ---------------------------------------
function readGraph() {
  try { return JSON.parse(fs.readFileSync(path.join(ROOT, "docs/graph/graph.json"), "utf8")); }
  catch { return null; }
}
const graph = readGraph();
for (const file of deletions) {
  if (!graph) { stop(`delete:${file}`, "graph unavailable — cannot prove nothing depends on it"); continue; }
  const ids = graph.nodes.filter((n) => n.source === file).map((n) => n.id);
  for (const id of ids) {
    const inbound = graph.edges.filter((e) => e.to === id);
    if (inbound.length) stop(`delete:${id}`, `${inbound.length} inbound edge(s), e.g. ${inbound[0].from}`);
  }
}

// --- destructive SQL --------------------------------------------------------
const DESTRUCTIVE = [
  [/\bdrop\s+(table|column|schema|type|index)\b/i, "DROP"],
  [/\btruncate\b/i, "TRUNCATE"],
  [/\balter\s+table\b[\s\S]{0,120}?\bdrop\b/i, "ALTER ... DROP"],
  [/\brename\s+(to|column)\b/i, "RENAME"],
  [/\bset\s+not\s+null\b/i, "NOT NULL on an existing column (needs a 3-step rollout)"],
];

// --- breaking proto changes -------------------------------------------------
// Removed lines that declare an rpc or a numbered field mean a removal or renumber.
const PROTO_BREAKING = [
  [/^-\s*rpc\s+\w+/, "removed or renamed rpc"],
  [/^-\s*(repeated\s+|optional\s+)?[\w.]+\s+\w+\s*=\s*\d+\s*;/, "removed, renamed or renumbered field"],
];

let currentFile = null;
for (const line of patch.split("\n")) {
  const header = line.match(/^\+\+\+ (?:b\/)?(.+)$/);
  if (header) { currentFile = header[1] === "/dev/null" ? null : header[1]; continue; }
  if (/^--- /.test(line)) continue;
  if (!currentFile) continue;

  if (/\.sql$/.test(currentFile) && line.startsWith("+")) {
    for (const [re, label] of DESTRUCTIVE) {
      if (re.test(line)) stop(`sql:${currentFile}`, `destructive statement (${label})`);
    }
  }
  if (/\.proto$/.test(currentFile) && line.startsWith("-") && !line.startsWith("---")) {
    for (const [re, label] of PROTO_BREAKING) {
      if (re.test(line)) stop(`proto:${currentFile}`, `${label} — breaking for installed clients`);
    }
  }
}

// --- report -----------------------------------------------------------------
const dedup = [...new Map(stops.map((s) => [`${s.rule}|${s.detail}`, s])).values()];
if (!dedup.length) {
  console.log(`AUTO — ${changed.length} file(s), nothing in the STOP column (docs/autonomy.md)`);
  process.exit(0);
}
console.error(`STOP — ${dedup.length} rule(s) hit (docs/autonomy.md):`);
for (const s of dedup) console.error(`  ${s.rule}: ${s.detail}`);
console.error("\nWrite an ESCALATIONS.md entry, mark the unit blocked, continue with other units.");
process.exit(3);
