import type { Member, Role } from "@fm/sdk/family/v1/family_pb";

/**
 * ROLE_ADMIN as its wire number, typed as the enum. The generated `Role` is a TypeScript
 * `enum`, and importing it as a *value* would make this module unloadable under `node --test`,
 * which strips types rather than compiling them — the same reason `noteBlocks.ts` writes its
 * block types out. The number comes from libs/proto/family/v1/family.proto and is fixed by the
 * contract: an enum value that changed number would already be a breaking wire change.
 */
const ROLE_ADMIN: Role = 1;

/**
 * Where the signed-in user stands inside their family.
 *
 * This lives here rather than in a screen because it decides whether DESTRUCTIVE controls are
 * drawn, and the app shell runs no tests — logic that gates "remove member" and "leave family"
 * should not be the one part nothing covers. It is also the same answer for every app: notes
 * and recipes need it as much as finance does.
 *
 * None of it is authorization. The server gates all four family-management calls itself, and
 * `userId` here comes from the access token's UNVERIFIED claims (see @fm/auth/claims.ts). This
 * decides what to render; the server decides what happens.
 */
export interface FamilyStanding {
  /** The caller's own membership row, or undefined when they are not in this list. */
  self: Member | undefined;
  /** Whether the caller may invite, rename, remove and see invitations. */
  isAdmin: boolean;
  /** How many admins the family has — what the last-admin rule is decided from. */
  adminCount: number;
  /**
   * Whether LeaveFamily would be accepted. The server refuses the last admin with
   * FailedPrecondition ("promote another admin before leaving"), so this is that rule
   * mirrored: an ordinary member may always leave, an admin may leave only if another remains.
   */
  canLeave: boolean;
}

export function familyStanding(
  members: readonly Member[] | undefined,
  userId: string | undefined,
): FamilyStanding {
  const roster = members ?? [];
  // An empty userId must never match a row. Claims are absent while anonymous, and a member
  // whose user_id somehow came back empty would otherwise be mistaken for the caller.
  const self = userId ? roster.find((member) => member.userId === userId) : undefined;
  const isAdmin = self?.role === ROLE_ADMIN;
  const adminCount = roster.filter((member) => member.role === ROLE_ADMIN).length;

  return {
    self,
    isAdmin,
    adminCount,
    // Not being in the roster at all means there is nothing to leave.
    canLeave: self !== undefined && (!isAdmin || adminCount > 1),
  };
}
