import type { Member, Role } from "@fm/sdk/family/v1/family_pb";

const ROLE_ADMIN: Role = 1;

export interface FamilyStanding {
  self: Member | undefined;
  isAdmin: boolean;
  adminCount: number;
  canLeave: boolean;
}

export function familyStanding(
  members: readonly Member[] | undefined,
  userId: string | undefined,
): FamilyStanding {
  const roster = members ?? [];
  const self = userId ? roster.find((member) => member.userId === userId) : undefined;
  const isAdmin = self?.role === ROLE_ADMIN;
  const adminCount = roster.filter((member) => member.role === ROLE_ADMIN).length;

  return {
    self,
    isAdmin,
    adminCount,
    canLeave: self !== undefined && (!isAdmin || adminCount > 1),
  };
}
