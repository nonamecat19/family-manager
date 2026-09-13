import assert from "node:assert/strict";
import { test } from "node:test";

import type { Member, Role as WireRole } from "@fm/sdk/family/v1/family_pb";

import { familyStanding } from "./family.ts";

const Role: Record<"UNSPECIFIED" | "ADMIN" | "MEMBER", WireRole> = {
  UNSPECIFIED: 0,
  ADMIN: 1,
  MEMBER: 2,
};

function member(userId: string, role: WireRole): Member {
  return { userId, familyId: "fam", displayName: userId, email: `${userId}@x`, role } as Member;
}

const admin = member("olena", Role.ADMIN);
const second = member("sergiy", Role.ADMIN);
const plain = member("taras", Role.MEMBER);

test("an admin is an admin", () => {
  const s = familyStanding([admin, plain], "olena");
  assert.equal(s.isAdmin, true);
  assert.equal(s.self?.userId, "olena");
  assert.equal(s.adminCount, 1);
});

test("an ordinary member is not", () => {
  const s = familyStanding([admin, plain], "taras");
  assert.equal(s.isAdmin, false);
  assert.equal(s.adminCount, 1);
});

test("an ordinary member may always leave", () => {
  assert.equal(familyStanding([admin, plain], "taras").canLeave, true);
});

test("the last admin may not leave", () => {
  const s = familyStanding([admin, plain], "olena");
  assert.equal(s.isAdmin, true);
  assert.equal(s.canLeave, false);
});

test("an admin may leave once a second admin exists", () => {
  const s = familyStanding([admin, second, plain], "olena");
  assert.equal(s.adminCount, 2);
  assert.equal(s.canLeave, true);
});

test("someone not in the roster has nothing to leave and no admin rights", () => {
  const s = familyStanding([admin, plain], "stranger");
  assert.equal(s.self, undefined);
  assert.equal(s.isAdmin, false);
  assert.equal(s.canLeave, false);
});

test("an absent user id never matches a row", () => {
  for (const id of [undefined, ""]) {
    const s = familyStanding([admin, plain], id);
    assert.equal(s.self, undefined, String(id));
    assert.equal(s.isAdmin, false, String(id));
    assert.equal(s.canLeave, false, String(id));
  }
});

test("an empty or missing roster is not an error", () => {
  for (const roster of [undefined, []]) {
    const s = familyStanding(roster, "olena");
    assert.equal(s.isAdmin, false);
    assert.equal(s.adminCount, 0);
    assert.equal(s.canLeave, false);
  }
});

test("an unspecified role is not treated as admin", () => {
  const s = familyStanding([member("olena", Role.UNSPECIFIED)], "olena");
  assert.equal(s.isAdmin, false);
  assert.equal(s.adminCount, 0);
  assert.equal(s.canLeave, true, "a non-admin may still leave");
});
