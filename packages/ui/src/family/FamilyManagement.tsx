import { useState } from "react";
import { Text, View } from "react-native";

import {
  Avatar,
  Badge,
  Button,
  Card,
  CardHeader,
  Divider,
  Field,
  InlineAction,
  Muted,
  Row,
  Title,
} from "../primitives.tsx";
import type { FamilyInvitationView, FamilyMemberView } from "./types.ts";
import { useTheme } from "../theme.tsx";

/**
 * The family-management surface, drawn once for every app.
 *
 * Every control here is gated server-side as well — UpdateFamily, RemoveMember,
 * RevokeInvitation and ListInvitations all require an admin, and RemoveMember refuses the
 * caller's own id. `isAdmin` therefore decides what to DRAW, never what is permitted: it comes
 * from the access token's unverified claims (see @fm/auth/claims.ts), so a tampered client
 * gets a rendered button and a refusal from the server, which is the correct outcome.
 *
 * Copy arrives as props. The library ships no strings, so an app can be Ukrainian-first
 * (finance) or English-first (recipes) without this file knowing either exists.
 */

/* ------------------------------------------------------------------ family name */

export interface FamilyNameCardStrings {
  heading: string;
  rename: string;
  save: string;
  cancel: string;
  nameLabel: string;
  /** Shown when the field is emptied — the server requires a name. */
  required: string;
}

export interface FamilyNameCardProps {
  name: string;
  isAdmin: boolean;
  strings: FamilyNameCardStrings;
  onRename: (name: string) => Promise<void>;
  /** Rendered under the field when the round trip fails. */
  error?: string | null;
  busy?: boolean;
}

export function FamilyNameCard({
  name,
  isAdmin,
  strings,
  onRename,
  error = null,
  busy = false,
}: FamilyNameCardProps) {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(name);
  const [localError, setLocalError] = useState<string | null>(null);

  const start = () => {
    // Seed from the current name each time rather than from the last aborted edit.
    setDraft(name);
    setLocalError(null);
    setEditing(true);
  };

  const submit = async () => {
    const next = draft.trim();
    if (next === "") {
      setLocalError(strings.required);
      return;
    }
    if (next === name) {
      setEditing(false);
      return;
    }
    await onRename(next);
    setEditing(false);
  };

  return (
    <Card>
      <CardHeader
        title={strings.heading}
        action={isAdmin && !editing ? <InlineAction label={strings.rename} onPress={start} /> : undefined}
      />
      <View style={{ paddingHorizontal: 16, paddingBottom: 14, gap: 12 }}>
        {editing ? (
          <>
            <Field
              label={strings.nameLabel}
              value={draft}
              onChangeText={(text) => {
                setDraft(text);
                setLocalError(null);
              }}
              autoFocus
              error={localError ?? error}
            />
            <View style={{ flexDirection: "row", gap: 8 }}>
              <View style={{ flex: 1 }}>
                <Button title={strings.save} onPress={() => void submit()} busy={busy} />
              </View>
              <View style={{ flex: 1 }}>
                <Button title={strings.cancel} tone="quiet" onPress={() => setEditing(false)} disabled={busy} />
              </View>
            </View>
          </>
        ) : (
          <Title>{name}</Title>
        )}
      </View>
    </Card>
  );
}

/* --------------------------------------------------------------------- members */

export interface FamilyMembersCardStrings {
  heading: string;
  you: string;
  admin: string;
  remove: string;
  empty: string;
}

export interface FamilyMembersCardProps {
  members: readonly FamilyMemberView[];
  isAdmin: boolean;
  strings: FamilyMembersCardStrings;
  onRemove: (member: FamilyMemberView) => void;
  /** The member currently being removed, so its row can show the pending state. */
  removingUserId?: string | null;
}

export function FamilyMembersCard({
  members,
  isAdmin,
  strings,
  onRemove,
  removingUserId = null,
}: FamilyMembersCardProps) {
  return (
    <Card>
      <CardHeader title={strings.heading} />
      {members.length === 0 ? (
        <View style={{ paddingHorizontal: 16, paddingBottom: 14 }}>
          <Muted>{strings.empty}</Muted>
        </View>
      ) : (
        members.map((member, index) => (
          <View key={member.userId}>
            {index > 0 ? <Divider /> : null}
            <Row>
              <Avatar name={member.displayName || member.email} index={index} />
              <View style={{ flex: 1, gap: 2 }}>
                <View style={{ flexDirection: "row", alignItems: "center", gap: 8 }}>
                  <Title>
                    {member.displayName || member.email}
                    {member.isSelf ? ` ${strings.you}` : ""}
                  </Title>
                  {member.role === "admin" ? <Badge label={strings.admin} tone="accent" /> : null}
                </View>
                <Muted>{member.email}</Muted>
              </View>
              {/* Removing yourself is LeaveFamily, a different call with a different guard —
                  the server rejects RemoveMember aimed at the caller, so it is not offered. */}
              {isAdmin && !member.isSelf ? (
                <InlineAction
                  label={strings.remove}
                  tone="danger"
                  onPress={() => onRemove(member)}
                  disabled={removingUserId === member.userId}
                />
              ) : null}
            </Row>
          </View>
        ))
      )}
    </Card>
  );
}

/* ----------------------------------------------------------------- invitations */

export interface FamilyInvitationsCardStrings {
  heading: string;
  revoke: string;
  empty: string;
  expired: string;
}

export interface FamilyInvitationsCardProps {
  invitations: readonly FamilyInvitationView[];
  strings: FamilyInvitationsCardStrings;
  onRevoke: (invitation: FamilyInvitationView) => void;
  revokingId?: string | null;
}

export function FamilyInvitationsCard({
  invitations,
  strings,
  onRevoke,
  revokingId = null,
}: FamilyInvitationsCardProps) {
  return (
    <Card>
      <CardHeader title={strings.heading} />
      {invitations.length === 0 ? (
        <View style={{ paddingHorizontal: 16, paddingBottom: 14 }}>
          <Muted>{strings.empty}</Muted>
        </View>
      ) : (
        invitations.map((invitation, index) => (
          <View key={invitation.id}>
            {index > 0 ? <Divider /> : null}
            <Row>
              <View style={{ flex: 1, gap: 2 }}>
                <Title>{invitation.email}</Title>
                {invitation.expiresLabel ? <Muted>{invitation.expiresLabel}</Muted> : null}
              </View>
              {/* Only a pending invitation can be revoked — the server answers anything else
                  with FailedPrecondition, so a spent one is shown as history, not an action. */}
              {invitation.pending ? (
                <InlineAction
                  label={strings.revoke}
                  tone="danger"
                  onPress={() => onRevoke(invitation)}
                  disabled={revokingId === invitation.id}
                />
              ) : (
                <Badge label={strings.expired} />
              )}
            </Row>
          </View>
        ))
      )}
    </Card>
  );
}

/* ---------------------------------------------------------------------- leaving */

export interface LeaveFamilyCardStrings {
  heading: string;
  body: string;
  leave: string;
  confirm: string;
  cancel: string;
  /** Why the button is unavailable — the server refuses the last admin. */
  lastAdmin: string;
}

export interface LeaveFamilyCardProps {
  strings: LeaveFamilyCardStrings;
  /** False when the caller is the only admin; the server would refuse, so it is explained. */
  canLeave: boolean;
  onLeave: () => void;
  busy?: boolean;
  error?: string | null;
}

export function LeaveFamilyCard({
  strings,
  canLeave,
  onLeave,
  busy = false,
  error = null,
}: LeaveFamilyCardProps) {
  const [confirming, setConfirming] = useState(false);

  return (
    <Card>
      <CardHeader title={strings.heading} />
      <View style={{ paddingHorizontal: 16, paddingBottom: 14, gap: 10 }}>
        <Muted>{canLeave ? strings.body : strings.lastAdmin}</Muted>
        {error ? <ErrorLine text={error} /> : null}
        {confirming ? (
          <View style={{ flexDirection: "row", gap: 8 }}>
            <View style={{ flex: 1 }}>
              <Button title={strings.confirm} tone="danger" onPress={onLeave} busy={busy} />
            </View>
            <View style={{ flex: 1 }}>
              <Button title={strings.cancel} tone="quiet" onPress={() => setConfirming(false)} disabled={busy} />
            </View>
          </View>
        ) : (
          <Button title={strings.leave} tone="danger" onPress={() => setConfirming(true)} disabled={!canLeave} />
        )}
      </View>
    </Card>
  );
}

function ErrorLine({ text }: { text: string }) {
  const t = useTheme();
  return <Text style={{ color: t.danger, fontSize: 13 }}>{text}</Text>;
}
