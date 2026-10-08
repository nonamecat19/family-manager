import { Annotation } from "@langchain/langgraph";

export interface BacklogUnit {
  id: string;
  title: string;
  kind: "contract" | "service" | "package" | "app" | "migration" | "test" | "docs" | "infra";
  nodes: string[];
  files: string[];
  depends_on: string[];
  gate: "auto" | "human";
  acceptance: string;
  status: "todo" | "doing" | "done" | "blocked";
  attempts: number;
  commit?: string | null;
  blocked_reason?: string | null;
}

export interface GateResult {
  rule: string;
  detail: string;
}

export interface VerifierFinding {
  severity: "blocker" | "risk" | "nit";
  path: string;
  line: number;
  problem: string;
  fix: string;
}

export interface HumanGateResponse {
  approved: boolean;
  reason?: string;
}

export const AutopilotState = Annotation.Root({
  briefId: Annotation<string>,
  baseSha: Annotation<string>,
  units: Annotation<BacklogUnit[]>,
  currentUnitId: Annotation<string | null>,
  currentUnit: Annotation<BacklogUnit | null>,
  gateResult: Annotation<GateResult[] | null>,
  verifierFindings: Annotation<VerifierFinding[]>,
  humanGateResponse: Annotation<HumanGateResponse | null>,
  runStatus: Annotation<"running" | "paused" | "complete" | "failed">,
  error: Annotation<string | null>,
});