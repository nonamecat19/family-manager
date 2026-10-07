#!/usr/bin/env node
import { Command } from "commander";
import { backlogBegin } from "./backlog.js";

const program = new Command();

program
  .name("orchestrator")
  .description("LangGraph-based autopilot for family-manager")
  .version("0.0.0");

program
  .command("autopilot <briefId>")
  .description("Run autonomous workflow for a brief")
  .option("--resume", "Resume from checkpoint")
  .action(async (briefId: string, _options: { resume?: boolean }) => {
    console.log(`[orchestrator] Starting autopilot for ${briefId}`);

    const baseSha = backlogBegin(briefId);
    console.log(`[orchestrator] Base SHA: ${baseSha}`);

    const initialState = {
      briefId,
      baseSha,
      units: [],
      currentUnitId: null,
      currentUnit: null,
      gateResult: null,
      verifierFindings: [],
      humanGateResponse: null,
      runStatus: "running" as const,
      error: null,
    };

    const config = {
      configurable: {
        thread_id: `autopilot-${briefId}`,
      },
    };

    try {
      const { autopilotGraph } = await import("./graph.js");
      const result = await autopilotGraph.invoke(initialState, config);
      console.log(`[orchestrator] Run completed: ${result.runStatus}`);
      if (result.error) {
        console.error(`[orchestrator] Error: ${result.error}`);
        process.exit(1);
      }
    } catch (e) {
      console.error(`[orchestrator] Fatal error:`, e);
      process.exit(1);
    }
  });

program
  .command("resume <briefId>")
  .description("Resume a paused autopilot run")
  .action(async (briefId: string) => {
    console.log(`[orchestrator] Resuming autopilot for ${briefId}`);

    const config = {
      configurable: {
        thread_id: `autopilot-${briefId}`,
      },
    };

    try {
      const { autopilotGraph } = await import("./graph.js");
      const result = await autopilotGraph.invoke(null, config);
      console.log(`[orchestrator] Resume completed: ${result.runStatus}`);
    } catch (e) {
      console.error(`[orchestrator] Resume failed:`, e);
      process.exit(1);
    }
  });

program.parse(process.argv);