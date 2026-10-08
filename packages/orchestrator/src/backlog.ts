import { execFileSync } from "node:child_process";
import path from "node:path";

const ROOT = path.resolve(import.meta.dirname, "../../..");

export function backlogCmd(...args: string[]): string {
  const result = execFileSync("node", ["tools/backlog/backlog.mjs", ...args], {
    cwd: ROOT,
    encoding: "utf8",
    maxBuffer: 32 << 20,
  });
  return result.trim();
}

export function backlogStatus(briefId: string): string {
  return backlogCmd("status", briefId);
}

export function backlogNext(briefId: string): string {
  return backlogCmd("next", briefId);
}

export function backlogStart(briefId: string, unitId: string): string {
  return backlogCmd("start", briefId, unitId);
}

export function backlogDone(briefId: string, unitId: string, commit: string): string {
  return backlogCmd("done", briefId, unitId, commit);
}

export function backlogFail(briefId: string, unitId: string, reason: string): string {
  return backlogCmd("fail", briefId, unitId, reason);
}

export function backlogBlock(briefId: string, unitId: string, reason: string): string {
  return backlogCmd("block", briefId, unitId, reason);
}

export function backlogUnblock(briefId: string, unitId: string): string {
  return backlogCmd("unblock", briefId, unitId);
}

export function backlogBegin(briefId: string): string {
  return backlogCmd("begin", briefId);
}

export interface ParsedUnit {
  id: string;
  title: string;
  kind: string;
  nodes: string[];
  files: string[];
  depends_on: string[];
  gate: "auto" | "human";
  status: "todo" | "doing" | "done" | "blocked";
  attempts: number;
  commit: string | null;
  blocked_reason: string | null;
  acceptance: string;
}

export function parseBacklogStatus(output: string): ParsedUnit[] {
  const lines = output.split("\n");
  const units: ParsedUnit[] = [];
  for (const line of lines) {
    const match = line.match(/^  \[(.)\] (\w+) (.+) \(([^,]+), gate:(\w+)(?: after:([^)]+))?\)(?:  <- (.*))?/);
    if (match) {
      const [, mark, id, title, kind, gate, dependsOn, blockedReason] = match;
      units.push({
        id: id ?? "",
        title: title ?? "",
        kind: kind ?? "",
        nodes: [],
        files: [],
        depends_on: dependsOn ? dependsOn.split(",") : [],
        gate: (gate ?? "auto") as "auto" | "human",
        status: mark === "x" ? "done" : mark === ">" ? "doing" : mark === "!" ? "blocked" : "todo",
        attempts: 0,
        commit: null,
        blocked_reason: blockedReason ?? null,
        acceptance: "",
      });
    }
  }
  return units;
}

export function getRunnableUnits(units: ParsedUnit[]): ParsedUnit[] {
  return units.filter(
    (u) => u.status === "todo" && u.depends_on.every((d) => units.find((x) => x.id === d)?.status === "done")
  );
}

export function getDoingUnit(units: ParsedUnit[]): ParsedUnit | null {
  return units.find((u) => u.status === "doing") ?? null;
}

export function isComplete(units: ParsedUnit[]): boolean {
  return units.every((u) => u.status === "done" || u.status === "blocked");
}