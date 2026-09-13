import { MemberRole, type Account, type Member, type QuickTemplate } from "@fm/api";

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

export function clockTime(epochMs: number): string {
  const at = new Date(epochMs);
  return `${String(at.getHours()).padStart(2, "0")}:${String(at.getMinutes()).padStart(2, "0")}`;
}
