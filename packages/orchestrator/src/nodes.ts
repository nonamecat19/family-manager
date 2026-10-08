import { Command } from "@langchain/langgraph";
import { AutopilotState } from "./state.js";
import { backlogStatus, backlogNext, backlogStart, backlogDone, backlogBlock, backlogFail, parseBacklogStatus, getRunnableUnits, getDoingUnit, isComplete } from "./backlog.js";
import { runGateCheck, runGateCheckRange } from "./gate-check.js";
import { runVerify, runGraphCheck } from "./verify.js";

export async function scopeNode(state: typeof AutopilotState.State) {
  const briefId = state.briefId;
  if (!briefId) {
    return new Command({ goto: "end", update: { error: "No briefId provided", runStatus: "failed" } });
  }

  const statusOutput = backlogStatus(briefId);
  const units = parseBacklogStatus(statusOutput);

  if (isComplete(units)) {
    return new Command({ goto: "end", update: { runStatus: "complete" } });
  }

  const doingUnit = getDoingUnit(units);
  if (doingUnit) {
    const currentUnit = units.find((u) => u.id === doingUnit.id);
    return new Command({
      goto: "implement",
      update: {
        units,
        currentUnitId: doingUnit.id,
        currentUnit: currentUnit ?? null,
      },
    });
  }

  const runnable = getRunnableUnits(units);
  if (runnable.length === 0) {
    return new Command({ goto: "end", update: { error: "No runnable units", runStatus: "failed" } });
  }

  const nextOutput = backlogNext(briefId);
  const nextUnit = JSON.parse(nextOutput);
  const currentUnit = units.find((u) => u.id === nextUnit.id);

  backlogStart(briefId, nextUnit.id);

  return new Command({
    goto: "implement",
    update: {
      units,
      currentUnitId: nextUnit.id,
      currentUnit: currentUnit ?? null,
    },
  });
}

export async function implementNode(state: typeof AutopilotState.State) {
  const unit = state.currentUnit;
  if (!unit) {
    return new Command({ goto: "scope", update: { error: "No current unit", runStatus: "failed" } });
  }

  console.log(`[autopilot] Implementing unit ${unit.id}: ${unit.title}`);
  console.log(`[autopilot] Acceptance: ${unit.acceptance}`);

  return new Command({
    goto: "acceptance",
    update: { units: state.units },
  });
}

export async function acceptanceNode(state: typeof AutopilotState.State) {
  const unit = state.currentUnit;
  if (!unit) {
    return new Command({ goto: "scope", update: { error: "No current unit", runStatus: "failed" } });
  }

  console.log(`[autopilot] Running acceptance for ${unit.id}: ${unit.acceptance}`);

  try {
    const { execFileSync } = await import("node:child_process");
    const ROOT = (await import("node:path")).resolve(import.meta.dirname, "../../..");
    const [cmd, ...args] = unit.acceptance.split(" ");
    if (!cmd) {
      throw new Error("Empty acceptance command");
    }
    execFileSync(cmd, args, {
      cwd: ROOT,
      encoding: "utf8",
      maxBuffer: 64 << 20,
      timeout: 300000,
    });
    console.log(`[autopilot] Acceptance passed for ${unit.id}`);
    return new Command({ goto: "verify" });
  } catch (e: unknown) {
    const error = e as { stdout?: string; stderr?: string };
    console.error(`[autopilot] Acceptance failed for ${unit.id}:`, error.stdout ?? error.stderr ?? e);
    return new Command({
      goto: "handleFailure",
      update: { error: String(error.stdout ?? error.stderr ?? e), units: state.units },
    });
  }
}

export async function verifyNode(state: typeof AutopilotState.State) {
  const unit = state.currentUnit;
  if (!unit) {
    return new Command({ goto: "scope", update: { error: "No current unit", runStatus: "failed" } });
  }

  console.log(`[autopilot] Running verify for ${unit.id}`);

  const graphCheck = runGraphCheck();
  if (!graphCheck.success) {
    console.error("[autopilot] Graph check failed:", graphCheck.output);
    return new Command({
      goto: "handleFailure",
      update: { error: `Graph check failed: ${graphCheck.output}`, units: state.units },
    });
  }

  const verifyResult = runVerify();
  if (!verifyResult.success) {
    console.error("[autopilot] Verify failed:", verifyResult.output);
    return new Command({
      goto: "handleFailure",
      update: { error: `Verify failed: ${verifyResult.output}`, units: state.units },
    });
  }

  console.log(`[autopilot] Verify passed for ${unit.id}`);
  return new Command({ goto: "gate", update: { units: state.units } });
}

export async function gateNode(state: typeof AutopilotState.State) {
  const unit = state.currentUnit;
  if (!unit) {
    return new Command({ goto: "scope", update: { error: "No current unit", runStatus: "failed" } });
  }

  console.log(`[autopilot] Running gate-check for ${unit.id}`);

  const gateResult = runGateCheck();
  if (!gateResult.auto) {
    console.log(`[autopilot] Gate STOP for ${unit.id}, requires human review`);
    if (unit.gate === "human") {
      return new Command({
        goto: "humanGate",
        update: { gateResult: gateResult.stops, units: state.units },
      });
    } else {
      return new Command({
        goto: "handleFailure",
        update: { error: `Unexpected gate stop: ${JSON.stringify(gateResult.stops)}`, units: state.units },
      });
    }
  }

  console.log(`[autopilot] Gate AUTO for ${unit.id}`);
  return new Command({ goto: "commit", update: { gateResult: [], units: state.units } });
}

export async function humanGateNode(state: typeof AutopilotState.State) {
  const unit = state.currentUnit;
  const stops = state.gateResult;

  console.log(`[autopilot] Human gate required for ${unit?.id}`);
  console.log("Stops:", JSON.stringify(stops, null, 2));

  return new Command({
    goto: "humanGate",
    update: { runStatus: "paused" },
  });
}

export async function commitNode(state: typeof AutopilotState.State) {
  const unit = state.currentUnit;
  const briefId = state.briefId;
  if (!unit || !briefId) {
    return new Command({ goto: "scope", update: { error: "Missing unit or briefId", runStatus: "failed" } });
  }

  console.log(`[autopilot] Committing unit ${unit.id}`);

  try {
    const { execFileSync } = await import("node:child_process");
    const ROOT = (await import("node:path")).resolve(import.meta.dirname, "../../..");

    execFileSync("git", ["add", "-A"], { cwd: ROOT, encoding: "utf8" });
    const gateCheckResult = runGateCheck();
    if (!gateCheckResult.auto) {
      return new Command({
        goto: "handleFailure",
        update: { error: `Gate check failed after verify: ${JSON.stringify(gateCheckResult.stops)}`, units: state.units },
      });
    }

    const commitMsg = `autopilot(${briefId}): ${unit.id} ${unit.title}`;
    execFileSync("git", ["commit", "-m", commitMsg], { cwd: ROOT, encoding: "utf8" });

    const shaOutput = execFileSync("git", ["rev-parse", "HEAD"], { cwd: ROOT, encoding: "utf8" }).trim();
    const sha = shaOutput.slice(0, 8);

    backlogDone(briefId, unit.id, sha);

    console.log(`[autopilot] Committed ${unit.id} @ ${sha}`);
    return new Command({ goto: "scope", update: { units: state.units, runStatus: "running" } });
  } catch (e: unknown) {
    const error = e as { stdout?: string; stderr?: string };
    console.error(`[autopilot] Commit failed for ${unit.id}:`, error.stdout ?? error.stderr ?? e);
    return new Command({
      goto: "handleFailure",
      update: { error: String(error.stdout ?? error.stderr ?? e), units: state.units },
    });
  }
}

export async function handleFailureNode(state: typeof AutopilotState.State) {
  const unit = state.currentUnit;
  const briefId = state.briefId;
  const error = state.error;

  if (!unit || !briefId) {
    return new Command({ goto: "end", update: { error: "No unit or briefId in failure handler", runStatus: "failed" } });
  }

  console.log(`[autopilot] Handling failure for ${unit.id}: ${error}`);

  const backlogOutput = backlogStatus(briefId);
  const units = parseBacklogStatus(backlogOutput);
  const currentUnit = units.find((u) => u.id === unit.id);

  if (currentUnit && currentUnit.attempts >= 2) {
    backlogBlock(briefId, unit.id, `3 failed attempts: ${error}`);
    console.log(`[autopilot] Unit ${unit.id} blocked after 3 attempts`);
    return new Command({ goto: "scope", update: { runStatus: "running" } });
  }

  console.log(`[autopilot] Retrying unit ${unit.id} (attempt ${(currentUnit?.attempts ?? 0) + 1}/3)`);
  backlogFail(briefId, unit.id, error ?? "unspecified");

  return new Command({ goto: "scope", update: { error: null, runStatus: "running" } });
}