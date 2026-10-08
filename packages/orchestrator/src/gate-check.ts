import { execFileSync } from "node:child_process";
import path from "node:path";

const ROOT = path.resolve(import.meta.dirname, "../../..");

export interface GateCheckResult {
  stops: GateStop[];
  auto: boolean;
}

export interface GateStop {
  rule: string;
  detail: string;
}

export function runGateCheck(range?: string): GateCheckResult {
  const args = range ? ["--range", range] : [];
  try {
    const output = execFileSync("node", ["tools/autonomy/gate-check.mjs", ...args], {
      cwd: ROOT,
      encoding: "utf8",
      maxBuffer: 32 << 20,
    });
    return { stops: [], auto: true };
  } catch (e: unknown) {
    const error = e as { stdout?: string; stderr?: string; status?: number };
    const output = error.stdout ?? error.stderr ?? "";
    if (error.status === 3) {
      const stops: GateStop[] = [];
      for (const line of output.split("\n")) {
        const match = line.match(/^  (.+): (.+)$/);
        if (match) {
          stops.push({ rule: match[1] ?? "", detail: match[2] ?? "" });
        }
      }
      return { stops, auto: false };
    }
    throw e;
  }
}

export function runGateCheckRange(range: string): GateCheckResult {
  return runGateCheck(range);
}