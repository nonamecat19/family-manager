import { StateGraph, START, interrupt, Command } from "@langchain/langgraph";
import { AutopilotState } from "./state.js";
import {
  scopeNode,
  implementNode,
  acceptanceNode,
  verifyNode,
  gateNode,
  commitNode,
  handleFailureNode,
} from "./nodes.js";

async function humanGateNodeWithInterrupt(state: typeof AutopilotState.State) {
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

export function createAutopilotGraph() {
  const graph = new StateGraph(AutopilotState)
    .addNode("scope", scopeNode)
    .addNode("implement", implementNode)
    .addNode("acceptance", acceptanceNode)
    .addNode("verify", verifyNode)
    .addNode("gate", gateNode)
    .addNode("humanGate", humanGateNodeWithInterrupt)
    .addNode("commit", commitNode)
    .addNode("handleFailure", handleFailureNode)
    .addEdge(START, "scope")
    .addEdge("implement", "acceptance")
    .addEdge("acceptance", "verify")
    .addEdge("verify", "gate")
    .addConditionalEdges("gate", (state) => {
      if (state.gateResult && state.gateResult.length > 0) {
        return "humanGate";
      }
      return "commit";
    })
    .addEdge("commit", "scope")
    .addEdge("handleFailure", "scope");

  return graph.compile();
}

export const autopilotGraph = createAutopilotGraph();