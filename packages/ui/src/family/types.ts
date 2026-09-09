/**
 * View models for the family screens.
 *
 * Deliberately NOT the generated SDK messages. `@fm/ui` depends on `@fm/theme` and `@fm/config`
 * and nothing else — pulling the SDK in would tie the component library to a contract version
 * and make a proto change a UI change. The app maps wire messages onto these; the mapping is
 * the app's job because the app is what knows which service it is talking to.
 */

export type FamilyRole = "admin" | "member";

export interface FamilyMemberView {
  userId: string;
  displayName: string;
  email: string;
  role: FamilyRole;
  /** Whether this row is the signed-in user. Drives "you" and hides self-removal. */
  isSelf: boolean;
}

export interface FamilyInvitationView {
  id: string;
  email: string;
  /** Only pending invitations can be revoked; the rest are shown as history. */
  pending: boolean;
  /** Preformatted by the app — this library does no date formatting or localization. */
  expiresLabel?: string;
}
