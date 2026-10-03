export type DeviceLoginPhase =
  | "idle"
  | "starting"
  | "pending"
  | "approved"
  | "denied"
  | "expired"
  | "error";

export const DEFAULT_POLL_INTERVAL_S = 5;
export const SLOW_DOWN_STEP_S = 5;

const TERMINAL: ReadonlySet<DeviceLoginPhase> = new Set(["approved", "denied", "expired", "error"]);

export function isTerminalPhase(phase: DeviceLoginPhase): boolean {
  return TERMINAL.has(phase);
}

export function pollIntervalSeconds(
  current: number,
  reply: { slowDown: boolean; intervalSeconds?: number | null },
): number {
  const base = current > 0 ? current : DEFAULT_POLL_INTERVAL_S;
  const offered = reply.intervalSeconds ?? 0;
  if (reply.slowDown) return Math.max(offered, base + SLOW_DOWN_STEP_S);
  return offered > 0 ? offered : base;
}

export function hasExpired(expiresAtMs: number | null | undefined, now: number): boolean {
  return expiresAtMs != null && now >= expiresAtMs;
}
