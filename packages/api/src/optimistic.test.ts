import assert from "node:assert/strict";
import { test } from "node:test";

import { bumpTemplateUsage, removeById } from "./optimistic.ts";

const templates = [
  { id: "t1", usageCount: 2 },
  { id: "t2", usageCount: 0 },
];

test("a template tap bumps only the tapped template", () => {
  const next = bumpTemplateUsage(templates, "t1");
  assert.deepEqual(next, [
    { id: "t1", usageCount: 3 },
    { id: "t2", usageCount: 0 },
  ]);
});

test("order is left alone — chips must not jump under the finger", () => {
  const next = bumpTemplateUsage(templates, "t2") ?? [];
  assert.deepEqual(
    next.map((t) => t.id),
    ["t1", "t2"],
  );
});

test("an unknown id returns the same reference, so nothing repaints", () => {
  assert.equal(bumpTemplateUsage(templates, "nope"), templates);
  assert.equal(bumpTemplateUsage(undefined, "t1"), undefined);
});

test("removeById drops one item and keeps the reference when it misses", () => {
  assert.deepEqual(removeById(templates, "t1"), [{ id: "t2", usageCount: 0 }]);
  assert.equal(removeById(templates, "nope"), templates);
  assert.equal(removeById(undefined, "t1"), undefined);
});
