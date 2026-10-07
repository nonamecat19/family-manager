import { describe, expect, it } from "vitest";
import { getDoingUnit, getRunnableUnits, isComplete, parseBacklogStatus } from "./backlog";

const STATUS = `0003-demo  base=abc123
done 1/5  doing 1  todo 2  blocked 1
  [x] u1 contract (with parens) (contract, gate:human)  deadbeef
  [>] u2 service impl (service, gate:auto after:u1)
  [ ] u3 app screen (app, gate:auto after:u1,u2)
  [ ] u4 docs (doc, gate:auto)
  [!] u5 migration (service, gate:human after:u1)  <- destructive SQL: DROP`;

describe("parseBacklogStatus", () => {
  const units = parseBacklogStatus(STATUS);

  it("parses every unit line and skips headers", () => {
    expect(units.map((u) => u.id)).toEqual(["u1", "u2", "u3", "u4", "u5"]);
  });

  it("maps marks to statuses", () => {
    expect(units.map((u) => u.status)).toEqual(["done", "doing", "todo", "todo", "blocked"]);
  });

  it("keeps parentheses inside titles", () => {
    expect(units[0]?.title).toBe("contract (with parens)");
    expect(units[0]?.kind).toBe("contract");
    expect(units[0]?.gate).toBe("human");
  });

  it("splits dependencies", () => {
    expect(units[2]?.depends_on).toEqual(["u1", "u2"]);
    expect(units[3]?.depends_on).toEqual([]);
  });

  it("captures the blocked reason", () => {
    expect(units[4]?.blocked_reason).toBe("destructive SQL: DROP");
    expect(units[3]?.blocked_reason).toBeNull();
  });
});

describe("unit selection", () => {
  const units = parseBacklogStatus(STATUS);

  it("runnable units are todo with all deps done", () => {
    expect(getRunnableUnits(units).map((u) => u.id)).toEqual(["u4"]);
  });

  it("finds the unit in progress", () => {
    expect(getDoingUnit(units)?.id).toBe("u2");
    expect(getDoingUnit(units.filter((u) => u.status !== "doing"))).toBeNull();
  });

  it("is complete only when every unit is done or blocked", () => {
    expect(isComplete(units)).toBe(false);
    expect(isComplete(units.filter((u) => u.status === "done" || u.status === "blocked"))).toBe(true);
  });
});
