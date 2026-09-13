export type FamilyRole = "admin" | "member";

export interface FamilyMemberView {
  userId: string;
  displayName: string;
  email: string;
  role: FamilyRole;
  isSelf: boolean;
}

export interface FamilyInvitationView {
  id: string;
  email: string;
  pending: boolean;
  expiresLabel?: string;
}
