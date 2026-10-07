import { SqliteSaver } from "@langchain/langgraph-checkpoint-sqlite";
import path from "node:path";
import fs from "node:fs";
import type { AutopilotState as AutopilotStateDef } from "./state.js";

const ROOT = path.resolve(import.meta.dirname, "../../..");
const DB_DIR = path.join(ROOT, ".orchestrator");
const DB_PATH = path.join(DB_DIR, "checkpoints.db");

let _checkpointer: ReturnType<typeof SqliteSaver.fromConnString> | null = null;

export function getCheckpointerSync() {
  if (!_checkpointer) {
    if (!fs.existsSync(DB_DIR)) {
      fs.mkdirSync(DB_DIR, { recursive: true });
    }
    _checkpointer = SqliteSaver.fromConnString(DB_PATH);
  }
  return _checkpointer;
}

export async function createAutopilotGraphWithCheckpointer() {
  const checkpointer = getCheckpointerSync();
  const { StateGraph, interrupt, Command } = await import("@langchain/langgraph");
  const { AutopilotState } = await import("./state.js");
  const { scopeNode, implementNode, acceptanceNode, verifyNode, gateNode, commitNode, handleFailureNode } = await import("./nodes.js");

  async function humanGateNodeWithInterrupt(state: typeof AutopilotStateDef.State) {
    const unit = state.currentUnit;
    const stops = state.gateResult;
    console.log(`[autopilot] Human gate required for ${unit?.id}`);
    console.log("Stops:", JSON.stringify(stops, null, 2));
    const response = await interrupt({
      type: "human_gate",
      unitId: unit?.id,
      stops,
      briefId: state.briefId,
    });
    return new Command({
      goto: response.approved ? "commit" : "handleFailure",
      update: {
        humanGateResponse: response,
        runStatus: response.approved ? "running" : "failed",
        error: response.approved ? null : `Human rejected: ${response.reason ?? "no reason given"}`,
      },
    });
  }

  const graph = new StateGraph(AutopilotState)
    .addNode("scope", scopeNode)
    .addNode("implement", implementNode)
    .addNode("acceptance", acceptanceNode)
    .addNode("verify", verifyNode)
    .addNode("gate", gateNode)
    .addNode("humanGate", humanGateNodeWithInterrupt)
    .addNode("commit", commitNode)
    .addNode("handleFailure", handleFailureNode)
    .addEdge("__start__", "scope")
    .addEdge("scope", "implement")
    .addEdge("implement", "acceptance")
    .addEdge("acceptance", "verify")
    .addEdge("verify", "gate")
    .addConditionalEdges("gate", (state: typeof AutopilotStateDef.State) => {
      if (state.gateResult && state.gateResult.length > 0) {
        return "humanGate";
      }
      return "commit";
    })
    .addEdge("commit", "scope")
    .addEdge("handleFailure", "scope")
    .addEdge("scope", "__end__");

  return graph.compile({ checkpointer });
}