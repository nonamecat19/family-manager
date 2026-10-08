import { execFileSync } from "node:child_process";
import path from "node:path";

const ROOT = path.resolve(import.meta.dirname, "../../..");

export interface VerifyResult {
  success: boolean;
  output: string;
}

export function runVerify(): VerifyResult {
  try {
    const output = execFileSync("just", ["verify"], {
      cwd: ROOT,
      encoding: "utf8",
      maxBuffer: 64 << 20,
      timeout: 300000,
    });
    return { success: true, output: output.trim() };
  } catch (e: unknown) {
    const error = e as { stdout?: string; stderr?: string; status?: number };
    return { success: false, output: (error.stdout ?? error.stderr ?? "").trim() };
  }
}

export function runCheckGo(): VerifyResult {
  try {
    const output = execFileSync("just", ["check-go"], {
      cwd: ROOT,
      encoding: "utf8",
      maxBuffer: 64 << 20,
      timeout: 300000,
    });
    return { success: true, output: output.trim() };
  } catch (e: unknown) {
    const error = e as { stdout?: string; stderr?: string; status?: number };
    return { success: false, output: (error.stdout ?? error.stderr ?? "").trim() };
  }
}

export function runCheckTs(): VerifyResult {
  try {
    const output = execFileSync("just", ["check-ts"], {
      cwd: ROOT,
      encoding: "utf8",
      maxBuffer: 64 << 20,
      timeout: 300000,
    });
    return { success: true, output: output.trim() };
  } catch (e: unknown) {
    const error = e as { stdout?: string; stderr?: string; status?: number };
    return { success: false, output: (error.stdout ?? error.stderr ?? "").trim() };
  }
}

export function runProtoCheck(): VerifyResult {
  try {
    const output = execFileSync("just", ["proto-check"], {
      cwd: ROOT,
      encoding: "utf8",
      maxBuffer: 64 << 20,
      timeout: 300000,
    });
    return { success: true, output: output.trim() };
  } catch (e: unknown) {
    const error = e as { stdout?: string; stderr?: string; status?: number };
    return { success: false, output: (error.stdout ?? error.stderr ?? "").trim() };
  }
}

export function runGraphCheck(): VerifyResult {
  try {
    const output = execFileSync("node", ["tools/repo-graph/extract.mjs", "--check"], {
      cwd: ROOT,
      encoding: "utf8",
      maxBuffer: 32 << 20,
    });
    return { success: true, output: output.trim() };
  } catch (e: unknown) {
    const error = e as { stdout?: string; stderr?: string; status?: number };
    return { success: false, output: (error.stdout ?? error.stderr ?? "").trim() };
  }
}