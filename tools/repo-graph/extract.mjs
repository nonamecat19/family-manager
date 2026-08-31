#!/usr/bin/env node
// Repo knowledge graph extractor.
// Stages 4-8 of docs/ontology.yaml: extract typed nodes/edges from the repo, validate
// domain/range, fuse duplicates, emit docs/graph/graph.json + graph.mmd.
//
//   node tools/repo-graph/extract.mjs              rebuild the graph
//   node tools/repo-graph/extract.mjs --impact ID  print blast radius of a node
//   node tools/repo-graph/extract.mjs --check      fail (exit 1) if the graph is stale
//
// No dependencies on purpose: this must run before `pnpm install` on a fresh clone.

import { execSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";

const ROOT = path.resolve(import.meta.dirname, "../..");
const OUT_DIR = path.join(ROOT, "docs/graph");
const EXTRACTED_AT = new Date().toISOString();
const COMMIT = sh("git rev-parse --short HEAD") || "unknown";

// --- ontology (mirror of docs/ontology.yaml; keep both in sync) --------------
const ENTITY_TYPES = new Set([
  "app", "package", "service", "golib", "gomodule", "contract",
  "rpc", "message", "table", "migration", "infra", "external",
]);
const RELATIONS = {
  DEPENDS_ON: { domain: ["app", "package", "service", "golib", "gomodule"], range: ["package", "golib", "gomodule", "external"] },
  IMPLEMENTS: { domain: ["service"], range: ["rpc", "contract"] },
  CONSUMES: { domain: ["app", "package", "service"], range: ["rpc", "contract"] },
  DECLARES: { domain: ["contract"], range: ["rpc", "message"] },
  USES_MESSAGE: { domain: ["rpc"], range: ["message"] },
  PERSISTS_TO: { domain: ["service"], range: ["table"] },
  CREATES: { domain: ["migration"], range: ["table"] },
  OWNED_BY: { domain: ["migration", "table", "rpc"], range: ["service"] },
  RUNS_ON: { domain: ["service"], range: ["infra"] },
  GENERATES: { domain: ["contract"], range: ["package", "golib"] },
};

const nodes = new Map();
const edges = [];
const dropped = [];

function addNode(id, type, props, prov) {
  if (!ENTITY_TYPES.has(type)) throw new Error(`unknown entity type ${type}`);
  const existing = nodes.get(id);
  if (existing) { Object.assign(existing.props, props); return existing; }
  const n = { id, type, props: props ?? {}, ...prov, extracted_at: EXTRACTED_AT, commit: COMMIT };
  nodes.set(id, n);
  return n;
}

function addEdge(from, rel, to, prov) {
  const spec = RELATIONS[rel];
  if (!spec) throw new Error(`unknown relation ${rel}`);
  edges.push({ from, rel, to, ...prov, extracted_at: EXTRACTED_AT, commit: COMMIT });
}

function sh(cmd) {
  try { return execSync(cmd, { cwd: ROOT, encoding: "utf8", stdio: ["ignore", "pipe", "ignore"] }).trim(); }
  catch { return ""; }
}

function rel(p) { return path.relative(ROOT, p); }
function read(p) { try { return fs.readFileSync(p, "utf8"); } catch { return null; } }
function lineOf(text, index) { return text.slice(0, index).split("\n").length; }
function dirs(p) {
  try { return fs.readdirSync(path.join(ROOT, p), { withFileTypes: true }).filter((d) => d.isDirectory()).map((d) => d.name); }
  catch { return []; }
}
function walk(dir, filter, acc = []) {
  let entries;
  try { entries = fs.readdirSync(dir, { withFileTypes: true }); } catch { return acc; }
  for (const e of entries) {
    if (["node_modules", ".git", "build", "dist", ".expo", ".dart_tool"].includes(e.name)) continue;
    const p = path.join(dir, e.name);
    if (e.isDirectory()) walk(p, filter, acc);
    else if (filter(p)) acc.push(p);
  }
  return acc;
}

// --- stage 4/5: TypeScript workspaces ---------------------------------------
function extractNode() {
  for (const [base, type] of [["apps", "app"], ["packages", "package"]]) {
    for (const name of dirs(base)) {
      const pkgPath = path.join(ROOT, base, name, "package.json");
      const raw = read(pkgPath);
      if (!raw) continue;
      const pkg = JSON.parse(raw);
      const id = type === "app" ? `app:${name}` : `pkg:${pkg.name ?? name}`;
      addNode(id, type, { dir: `${base}/${name}`, name: pkg.name ?? name }, { source: rel(pkgPath) });
      const deps = { ...(pkg.dependencies ?? {}), ...(pkg.devDependencies ?? {}) };
      for (const [dep, range] of Object.entries(deps)) {
        const internal = String(range).startsWith("workspace:");
        const target = internal ? `pkg:${dep}` : `ext:${dep}`;
        if (!internal) addNode(target, "external", { name: dep }, { source: rel(pkgPath), confidence: 0.9 });
        addEdge(id, "DEPENDS_ON", target, { source: rel(pkgPath), internal });
      }
    }
  }
  const sdk = read(path.join(ROOT, "sdk/typescript/package.json"));
  if (sdk) {
    const pkg = JSON.parse(sdk);
    addNode(`pkg:${pkg.name}`, "package", { dir: "sdk/typescript", generated: true }, { source: "sdk/typescript/package.json" });
  }
}

// --- stage 4/5: Go modules ---------------------------------------------------
function goModuleId(dir) {
  const raw = read(path.join(ROOT, dir, "go.mod"));
  if (!raw) return null;
  const m = raw.match(/^module\s+(\S+)/m);
  return m ? m[1] : null;
}

function extractGo() {
  const work = read(path.join(ROOT, "go.work")) ?? "";
  const uses = [...work.matchAll(/^\s*(?:use\s+)?(\.\/[^\s)]+)/gm)].map((m) => m[1].replace(/^\.\//, ""));
  const localModules = new Map(); // module path -> node id

  for (const dir of uses) {
    const modPath = goModuleId(dir);
    if (!modPath) { dropped.push({ why: "go.work member without go.mod", dir }); continue; }
    // fusion: directory location decides the entity type, module path is the identity
    let type = "gomodule";
    if (dir.startsWith("services/")) type = "service";
    else if (dir.startsWith("libs/go/") || dir === "sdk/go") type = "golib";
    else if (dir.endsWith("-service")) type = "service"; // pre-migration layout
    const name = path.basename(dir).replace(/-service$/, "");
    const id = type === "service" ? `service:${name}` : type === "golib" ? `golib:${name}` : `gomod:${modPath}`;
    addNode(id, type, { dir, module: modPath, aliases: [modPath, dir, name] }, { source: `${dir}/go.mod` });
    localModules.set(modPath, id);
  }

  for (const [modPath, id] of localModules) {
    const dir = nodes.get(id).props.dir;
    const raw = read(path.join(ROOT, dir, "go.mod")) ?? "";
    for (const m of raw.matchAll(/^\s+(\S+)\s+v\S+(\s+\/\/ indirect)?$/gm)) {
      if (m[2]) continue; // indirect deps stay out of the graph — noise, not architecture
      const target = localModules.get(m[1]) ?? `ext:${m[1]}`;
      if (!localModules.has(m[1])) addNode(target, "external", { name: m[1] }, { source: `${dir}/go.mod`, confidence: 0.9 });
      addEdge(id, "DEPENDS_ON", target, { source: `${dir}/go.mod`, line: lineOf(raw, m.index), internal: localModules.has(m[1]) });
    }
  }
  return localModules;
}

// --- stage 4/5: proto contracts ---------------------------------------------
function extractProto(localModules) {
  const protoFiles = [
    ...walk(path.join(ROOT, "libs/proto"), (p) => p.endsWith(".proto")),
    ...walk(path.join(ROOT, "services"), (p) => p.endsWith(".proto")),
    ...dirs(".").filter((d) => d.endsWith("-service")).flatMap((d) => walk(path.join(ROOT, d), (p) => p.endsWith(".proto"))),
  ];
  for (const file of protoFiles) {
    const text = read(file);
    if (!text) continue;
    const source = rel(file);
    const contractId = `proto:${source}`;
    const pkg = text.match(/^package\s+([\w.]+)\s*;/m)?.[1] ?? "unknown";
    const goPkg = text.match(/option\s+go_package\s*=\s*"([^"]+)"/)?.[1] ?? null;
    addNode(contractId, "contract", { package: pkg, go_package: goPkg, canonical: source.startsWith("libs/proto") }, { source });

    for (const svc of text.matchAll(/service\s+(\w+)\s*\{([\s\S]*?)\n\}/g)) {
      const svcName = svc[1];
      for (const m of svc[2].matchAll(/rpc\s+(\w+)\s*\(\s*(?:stream\s+)?([\w.]+)\s*\)\s*returns\s*\(\s*(?:stream\s+)?([\w.]+)\s*\)/g)) {
        const rpcId = `rpc:${svcName}.${m[1]}`;
        const prov = { source, line: lineOf(text, svc.index + m.index) };
        addNode(rpcId, "rpc", { service: svcName, name: m[1] }, prov);
        addEdge(contractId, "DECLARES", rpcId, prov);
        for (const msg of [m[2], m[3]]) {
          const msgId = `msg:${msg.includes(".") ? msg : `${pkg}.${msg}`}`;
          addNode(msgId, "message", { name: msg }, prov);
          addEdge(rpcId, "USES_MESSAGE", msgId, prov);
        }
      }
    }
    for (const m of text.matchAll(/^message\s+(\w+)\s*\{/gm)) {
      addNode(`msg:${pkg}.${m[1]}`, "message", { name: m[1] }, { source, line: lineOf(text, m.index) });
    }

    // fusion: bind the contract to the service that implements it.
    // 1. libs/proto/<domain>/v1/*.proto  -> service:<domain>            (convention, confidence 1)
    // 2. go_package pointing at a service module                        (confidence 1)
    // 3. contract sitting inside a service directory                    (confidence 0.6)
    // sdk/go and libs/go modules are generated/shared — never owners.
    let owner = null;
    let conf = 1;
    const domain = source.startsWith("libs/proto/") ? source.split("/")[2] : null;
    if (domain && nodes.has(`service:${domain}`)) owner = `service:${domain}`;
    if (!owner && goPkg) {
      for (const [modPath, id] of localModules) {
        if (nodes.get(id)?.type === "service" && goPkg.startsWith(modPath)) owner = id;
      }
    }
    if (!owner) {
      const top = source.split("/")[0];
      owner = [...nodes.values()].find((n) => n.type === "service" && n.props.dir === top)?.id ?? null;
      conf = 0.6;
    }
    if (owner) {
      addEdge(owner, "IMPLEMENTS", contractId, { source, confidence: conf });
      for (const n of nodes.values()) if (n.type === "rpc" && n.source === source) addEdge(n.id, "OWNED_BY", owner, { source });
    }
  }
}

// --- stage 4/5: persistence --------------------------------------------------
function extractSql() {
  const owners = [...nodes.values()].filter((n) => n.type === "service");
  const sqlFiles = owners.flatMap((svc) => walk(path.join(ROOT, svc.props.dir), (p) => p.endsWith(".sql")).map((f) => [svc, f]));
  for (const [svc, file] of sqlFiles) {
    const text = read(file);
    if (!text) continue;
    const source = rel(file);
    const isMigration = source.includes("/migrations/");
    if (isMigration) addNode(`migration:${source}`, "migration", {}, { source });
    for (const m of text.matchAll(/create\s+table\s+(?:if\s+not\s+exists\s+)?"?(\w+)"?/gi)) {
      const t = `table:${m[1]}`;
      const prov = { source, line: lineOf(text, m.index) };
      addNode(t, "table", { name: m[1] }, prov);
      if (isMigration) { addEdge(`migration:${source}`, "CREATES", t, prov); addEdge(`migration:${source}`, "OWNED_BY", svc.id, prov); }
      addEdge(svc.id, "PERSISTS_TO", t, prov);
      addEdge(t, "OWNED_BY", svc.id, prov);
    }
    for (const m of text.matchAll(/\b(?:from|join|into|update)\s+"?(\w+)"?/gi)) {
      const t = `table:${m[1]}`;
      if (!nodes.has(t)) continue; // only edges to tables we actually saw created
      addEdge(svc.id, "PERSISTS_TO", t, { source, line: lineOf(text, m.index), confidence: 0.8 });
    }
  }
}

// --- stage 4/5: infra --------------------------------------------------------
function extractInfra() {
  const text = read(path.join(ROOT, "docker-compose.yml"));
  if (!text) return;
  const block = text.split(/^volumes:/m)[0];
  for (const m of block.matchAll(/^ {2}(\w[\w-]*):$/gm)) {
    addNode(`infra:${m[1]}`, "infra", { name: m[1] }, { source: "docker-compose.yml", line: lineOf(text, m.index) });
  }
  for (const svc of [...nodes.values()].filter((n) => n.type === "service")) {
    const envs = walk(path.join(ROOT, svc.props.dir), (p) => path.basename(p).startsWith(".env")).map(read).join("\n");
    for (const infra of [...nodes.values()].filter((n) => n.type === "infra")) {
      if (new RegExp(infra.props.name, "i").test(envs)) {
        addEdge(svc.id, "RUNS_ON", infra.id, { source: `${svc.props.dir}/.env.example`, confidence: 0.7 });
      }
    }
  }
}

// --- stage 7: validate domain/range, drop violations -------------------------
function validate() {
  const kept = [];
  const seen = new Set();
  for (const e of edges) {
    const from = nodes.get(e.from), to = nodes.get(e.to);
    const spec = RELATIONS[e.rel];
    if (!from || !to) { dropped.push({ why: "dangling endpoint", edge: e }); continue; }
    if (!spec.domain.includes(from.type) || !spec.range.includes(to.type)) { dropped.push({ why: "domain/range", edge: e }); continue; }
    const key = `${e.from}|${e.rel}|${e.to}`;
    if (seen.has(key)) continue; // fusion: identical edges collapse, first provenance wins
    seen.add(key);
    kept.push(e);
  }
  return kept;
}

// --- serve: mermaid + impact -------------------------------------------------
function mermaid(graph) {
  const shown = graph.nodes.filter((n) => ["app", "package", "service", "golib", "contract", "table", "infra"].includes(n.type));
  const ids = new Set(shown.map((n) => n.id));
  const safe = (id) => id.replace(/[^\w]/g, "_");
  const lines = ["graph LR"];
  for (const t of ["app", "package", "service", "golib", "contract", "table", "infra"]) {
    const group = shown.filter((n) => n.type === t);
    if (!group.length) continue;
    lines.push(`  subgraph ${t}s`);
    for (const n of group) lines.push(`    ${safe(n.id)}["${n.id}"]`);
    lines.push("  end");
  }
  for (const e of graph.edges) {
    if (!ids.has(e.from) || !ids.has(e.to)) continue;
    lines.push(`  ${safe(e.from)} -->|${e.rel}| ${safe(e.to)}`);
  }
  return lines.join("\n") + "\n";
}

function impact(graph, target) {
  const back = new Map();
  for (const e of graph.edges) {
    if (!back.has(e.to)) back.set(e.to, []);
    back.get(e.to).push(e);
  }
  const seen = new Set([target]);
  const out = [];
  const queue = [[target, 0]];
  while (queue.length) {
    const [id, depth] = queue.shift();
    for (const e of back.get(id) ?? []) {
      if (seen.has(e.from)) continue;
      seen.add(e.from);
      out.push({ node: e.from, via: `${e.rel} -> ${id}`, depth: depth + 1, source: e.source });
      queue.push([e.from, depth + 1]);
    }
  }
  return out;
}

// --- main --------------------------------------------------------------------
const args = process.argv.slice(2);
extractNode();
const localModules = extractGo();
extractProto(localModules);
extractSql();
extractInfra();

const graph = {
  version: 1,
  generated_at: EXTRACTED_AT,
  commit: COMMIT,
  nodes: [...nodes.values()].sort((a, b) => a.id.localeCompare(b.id)),
  edges: validate().sort((a, b) => (a.from + a.rel + a.to).localeCompare(b.from + b.rel + b.to)),
  dropped,
};

const impactArg = args.indexOf("--impact");
if (impactArg !== -1) {
  const target = args[impactArg + 1];
  const existing = JSON.parse(read(path.join(OUT_DIR, "graph.json")) ?? "null") ?? graph;
  if (!existing.nodes.some((n) => n.id === target)) {
    console.error(`node not found: ${target}\ncandidates:\n${existing.nodes.map((n) => "  " + n.id).join("\n")}`);
    process.exit(2);
  }
  const rows = impact(existing, target);
  console.log(`blast radius of ${target}: ${rows.length} node(s)`);
  for (const r of rows) console.log(`  [${r.depth}] ${r.node}  (${r.via})  ${r.source}`);
  process.exit(0);
}

fs.mkdirSync(OUT_DIR, { recursive: true });
const json = JSON.stringify(graph, null, 2) + "\n";
if (args.includes("--check")) {
  const prev = read(path.join(OUT_DIR, "graph.json"));
  // `commit` is stripped alongside the timestamps: it records the HEAD the facts were read
  // at, so committing the graph changes it and the very next --check would call the file
  // stale — with no way to ever satisfy it. Staleness is about structure (nodes, edges,
  // sources), which the rest of the comparison covers.
  const strip = (s) => (s ?? "").replace(/"(generated_at|extracted_at|commit)": "[^"]*"/g, "");
  if (strip(prev) !== strip(json)) { console.error("graph is stale — run `just graph`"); process.exit(1); }
  console.log("graph up to date");
  process.exit(0);
}
fs.writeFileSync(path.join(OUT_DIR, "graph.json"), json);
fs.writeFileSync(path.join(OUT_DIR, "graph.mmd"), mermaid(graph));
console.log(`nodes=${graph.nodes.length} edges=${graph.edges.length} dropped=${dropped.length} -> docs/graph/`);
