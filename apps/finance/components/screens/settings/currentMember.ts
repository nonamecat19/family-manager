import { MemberRole, type Account, type Member, type QuickTemplate } from "@fm/api";

/**
 * Who is signed in.
 *
 * There is no "current user" RPC and no claim exposed by @fm/auth (the token never leaves
 * `SessionManager`), so this screen derives it from two facts the server only ever scopes to
 * the caller:
 *
 *   - `ListTemplates` returns the CALLER's templates (the handler ignores any other
 *     owner_user_id), so a template's `ownerUserId` is the caller;
 *   - `ListAccounts.privateOwn` is, by the same rule, the caller's own private accounts, so
 *     `ownerMemberId` is the caller.
 *
 * A household member with neither falls back to the owner, then to the first member — the
 * drawer still needs a name to draw. Returns null only for an empty member list.
 */
export function currentMember(
  members: readonly Member[] | undefined,
  templates: readonly QuickTemplate[] | undefined,
  privateOwn: readonly Account[] | undefined,
): Member | null {
  if (!members || members.length === 0) return null;

  const ownerUserId =
    templates?.find((template) => template.ownerUserId !== "")?.ownerUserId ??
    privateOwn?.find((account) => account.ownerMemberId !== "")?.ownerMemberId;

  const identified = ownerUserId
    ? members.find((member) => member.userId === ownerUserId)
    : undefined;

  return identified ?? members.find((member) => member.role === MemberRole.OWNER) ?? members[0] ?? null;
}

/** "11:38" — the drawer's footer stamp, from a query's `dataUpdatedAt`. */
export function clockTime(epochMs: number): string {
  const at = new Date(epochMs);
  return `${String(at.getHours()).padStart(2, "0")}:${String(at.getMinutes()).padStart(2, "0")}`;
}
