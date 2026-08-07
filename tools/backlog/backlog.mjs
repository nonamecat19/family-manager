#!/usr/bin/env node
// Durable backlog for autonomous runs. State lives in docs/backlog/<brief>.json, NOT in an
// agent's context — a run that loses its session re-reads the file and continues.
// The state machine is deterministic here so the model never has to remember where it was.
//
//   backlog.mjs new <brief-id> <branch>          create an empty backlog
//   backlog.mjs add <brief-id> '<unit json>'     append a unit
//   backlog.mjs status <brief-id>                human/agent readable summary
//   backlog.mjs next <brief-id>                  print the next ready unit (exit 4 if none)
//   backlog.mjs start <brief-id> <unit-id>
//   backlog.mjs done <brief-id> <unit-id> [sha]
//   backlog.mjs block <brief-id> <unit-id> "reason"
//   backlog.mjs unblock <brief-id> <unit-id>
//   backlog.mjs fail <brief-id> <unit-id> "reason"   increments attempts; blocks at 3
//
// A unit: { id, title, kind, nodes[], files[], depends_on[], acceptance, gate, status,
//           attempts, commit, blocked_reason }
// status: todo | doing | done | blocked

import fs from "node:fs";
import path from "node:path";

const ROOT = path.resolve(import.meta.dirname, "../..");
const DIR = path.join(ROOT, "docs/backlog");
const MAX_ATTEMPTS = 3;

const [cmd, brief, ...rest] = process.argv.slice(2);
if (!cmd) die("usage: backlog.mjs <new|add|status|next|start|done|block|unblock|fail> ...");

function die(msg, code = 1) { console.error(msg); process.exit(code); }
function file(id) { return path.join(DIR, `${id}.json`); }
function load(id) {
  try { return JSON.parse(fs.readFileSync(file(id), "utf8")); }
  catch { die(`no backlog for brief "${id}" — run: backlog.mjs new ${id} <branch>`); }
}
function save(b) {
  b.updated_at = new Date().toISOString();
  fs.mkdirSync(DIR, { recursive: true });
  fs.writeFileSync(file(b.brief), JSON.stringify(b, null, 2) + "\n");
}
function unit(b, id) {
  const u = b.units.find((x) => x.id === id);
  if (!u) die(`no unit "${id}" in ${b.brief}`);
  return u;
}

switch (cmd) {
  case "new": {
    const branch = rest[0] ?? `auto/${brief}`;
    if (fs.existsSync(file(brief))) die(`backlog ${brief} already exists`);
    save({ brief, branch, created_at: new Date().toISOString(), units: [] });
    console.log(`created docs/backlog/${brief}.json on branch ${branch}`);
    break;
  }

  case "add": {
    const b = load(brief);
    let u;
    try { u = JSON.parse(rest[0]); } catch { die("unit must be valid JSON"); }
    for (const k of ["id", "title", "kind", "acceptance"]) if (!u[k]) die(`unit is missing "${k}"`);
    if (b.units.some((x) => x.id === u.id)) die(`unit ${u.id} already exists`);
    b.units.push({
      nodes: [], files: [], depends_on: [], gate: "auto",
      status: "todo", attempts: 0, commit: null, blocked_reason: null, ...u,
    });
    save(b);
    console.log(`added ${u.id}: ${u.title}`);
    break;
  }

  case "status": {
    const b = load(brief);
    const by = (s) => b.units.filter((u) => u.status === s);
    console.log(`${b.brief}  branch=${b.branch}`);
    console.log(`done ${by("done").length}/${b.units.length}  doing ${by("doing").length}  todo ${by("todo").length}  blocked ${by("blocked").length}`);
    for (const u of b.units) {
      const mark = { done: "x", doing: ">", blocked: "!", todo: " " }[u.status];
      const dep = u.depends_on.length ? ` after:${u.depends_on.join(",")}` : "";
      const extra = u.status === "blocked" ? `  <- ${u.blocked_reason}` : u.commit ? `  ${u.commit}` : "";
      console.log(`  [${mark}] ${u.id} ${u.title} (${u.kind}, gate:${u.gate}${dep})${extra}`);
    }
    const runnable = b.units.some((u) => u.status === "todo" && u.depends_on.every((d) => unit(b, d).status === "done"));
    console.log(runnable ? "RUNNABLE: yes" : "RUNNABLE: no — every remaining unit is blocked or waiting on a blocked dependency");
    break;
  }

  case "next": {
    const b = load(brief);
    const doing = b.units.find((u) => u.status === "doing");
    if (doing) { console.log(JSON.stringify(doing, null, 2)); break; } // resume, do not start a second
    const ready = b.units.find(
      (u) => u.status === "todo" && u.depends_on.every((d) => unit(b, d).status === "done"),
    );
    if (!ready) {
      const blocked = b.units.filter((u) => u.status === "blocked").length;
      const left = b.units.filter((u) => u.status !== "done").length;
      console.error(left === 0 ? "COMPLETE" : `NO READY UNIT — ${left} remaining, ${blocked} blocked`);
      process.exit(4);
    }
    console.log(JSON.stringify(ready, null, 2));
    break;
  }

  case "start": {
    const b = load(brief); const u = unit(b, rest[0]);
    const unmet = u.depends_on.filter((d) => unit(b, d).status !== "done");
    if (unmet.length) die(`cannot start ${u.id}: unmet dependencies ${unmet.join(", ")}`);
    u.status = "doing"; save(b);
    console.log(`started ${u.id} (attempt ${u.attempts + 1}/${MAX_ATTEMPTS})`);
    break;
  }

  case "done": {
    const b = load(brief); const u = unit(b, rest[0]);
    u.status = "done"; u.commit = rest[1] ?? null; u.blocked_reason = null; save(b);
    const left = b.units.filter((x) => x.status !== "done" && x.status !== "blocked").length;
    console.log(`done ${u.id}${u.commit ? ` @ ${u.commit}` : ""} — ${left} unit(s) left`);
    break;
  }

  case "fail": {
    const b = load(brief); const u = unit(b, rest[0]);
    u.attempts += 1;
    if (u.attempts >= MAX_ATTEMPTS) {
      u.status = "blocked";
      u.blocked_reason = `${MAX_ATTEMPTS} failed attempts: ${rest[1] ?? "unspecified"}`;
      save(b);
      console.error(`BLOCKED ${u.id} after ${MAX_ATTEMPTS} attempts — escalate, the approach is wrong`);
      process.exit(3);
    }
    u.status = "todo"; save(b);
    console.log(`attempt ${u.attempts}/${MAX_ATTEMPTS} failed for ${u.id} — retry with a different approach`);
    break;
  }

  case "block": {
    const b = load(brief); const u = unit(b, rest[0]);
    u.status = "blocked"; u.blocked_reason = rest[1] ?? "gate";
    save(b);
    const orphaned = b.units.filter((x) => x.depends_on.includes(u.id) && x.status === "todo").map((x) => x.id);
    console.log(`blocked ${u.id}: ${u.blocked_reason}`);
    if (orphaned.length) console.log(`  waiting on it: ${orphaned.join(", ")} — these cannot run either`);
    break;
  }

  case "unblock": {
    const b = load(brief); const u = unit(b, rest[0]);
    u.status = "todo"; u.blocked_reason = null; u.attempts = 0; save(b);
    console.log(`unblocked ${u.id}`);
    break;
  }

  default:
    die(`unknown command "${cmd}"`);
}
