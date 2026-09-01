import { useFamily, useShareNote, useShareNotebook, useShares, useUnshare } from "@fm/api";
import type { Member } from "@fm/sdk/family/v1/family_pb";
import { SharePermission, ShareSubject, type Share } from "@fm/sdk/notes/v1/notes_pb";
import { useCallback, useMemo } from "react";
import { Modal, Pressable, ScrollView, Text, View } from "react-native";

import { strings } from "./i18n/index.ts";
import { Avatar, Divider, Icon, IconButton, nocturne } from "./nocturne/index.ts";

/**
 * Sharing, in one sheet.
 *
 * The product rule this screen exists to express: a note is PRIVATE until someone shares it.
 * So the sheet opens on "not shared" and every row is an explicit grant — to one family
 * member, or to the whole family — at VIEW or EDIT. There is no comment-only tier; the proto
 * says why, and adding a third state in the UI would be inventing a rule the server does not
 * enforce.
 *
 * Family members come from the existing @fm/api family hooks. This app never asks the notes
 * service who is in the family: the notes service does not know, it trusts the token's claim.
 */

/** The family as a name lookup. Everything that prints a person's name goes through it. */
export function useMemberDirectory() {
  const family = useFamily();
  const members: Member[] = useMemo(() => family.data?.members ?? [], [family.data]);

  const nameOf = useCallback(
    (userId: string): string => {
      const member = members.find((m) => m.userId === userId);
      return member?.displayName ?? member?.email ?? "";
    },
    [members],
  );

  return { members, nameOf, isPending: family.isPending };
}

export interface ShareSheetProps {
  visible: boolean;
  onClose: () => void;
  /** Exactly one of these, matching where the share is granted. */
  noteId?: string;
  notebookId?: string;
  /** The owner is drawn as a row that cannot be revoked. */
  ownerUserId?: string;
}

export function ShareSheet({ visible, onClose, noteId, notebookId, ownerUserId }: ShareSheetProps) {
  const target = notebookId ? { notebookId } : { noteId: noteId ?? "" };
  const shares = useShares(target);
  const { members } = useMemberDirectory();
  const shareNote = useShareNote();
  const shareNotebook = useShareNotebook();
  const unshare = useUnshare();

  const rows = shares.data ?? [];
  const familyShare = rows.find((s) => s.subject === ShareSubject.FAMILY);

  const grant = (subject: ShareSubject, memberUserId: string, permission: SharePermission) => {
    if (notebookId) {
      shareNotebook.mutate({ notebookId, subject, memberUserId, permission });
      return;
    }
    if (noteId) shareNote.mutate({ noteId, subject, memberUserId, permission });
  };

  const revoke = (share: Share) => {
    unshare.mutate({ ...target, shareId: share.id });
  };

  const busy = shareNote.isPending || shareNotebook.isPending || unshare.isPending;

  return (
    <Modal visible={visible} transparent animationType="slide" onRequestClose={onClose}>
      <Pressable
        accessibilityRole="button"
        accessibilityLabel={strings.common.close}
        onPress={onClose}
        className="flex-1"
        style={SCRIM}
      />
      <View
        className="max-h-[70%] gap-[14px] bg-surface px-[18px] pb-[24px] pt-[10px]"
        style={{ borderTopLeftRadius: 20, borderTopRightRadius: 20 }}
      >
        <View className="h-[4px] w-[36px] self-center rounded-sm bg-neutral-800" />

        <View className="flex-row items-center">
          <Text className="font-med text-[16px] text-fg">
            {notebookId ? strings.share.titleNotebook : strings.share.title}
          </Text>
          <View className="ml-auto">
            <IconButton icon="x" label={strings.common.close} onPress={onClose} size={17} color={nocturne.neutral[500]} />
          </View>
        </View>

        <Text className="font-sans text-[12.5px] leading-[19px] text-neutral-500">
          {strings.share.body}
        </Text>

        <ScrollView className="grow-0">
          <View className="gap-[4px]">
            <ShareRow
              label={strings.share.wholeFamily}
              icon="users-three"
              permission={familyShare?.permission}
              disabled={busy}
              onSet={(permission) => grant(ShareSubject.FAMILY, "", permission)}
              onRemove={familyShare ? () => revoke(familyShare) : undefined}
            />

            <View className="py-[6px]">
              <Divider />
            </View>

            {members.length === 0 ? (
              <Text className="py-[8px] font-sans text-[12.5px] text-neutral-500">
                {strings.share.noMembers}
              </Text>
            ) : null}

            {members.map((member) => {
              if (member.userId === ownerUserId) {
                return (
                  <View key={member.userId} className="flex-row items-center gap-[9px] py-[9px]">
                    <Avatar name={member.displayName || member.email} size={24} />
                    <Text className="font-sans text-[13.5px] text-fg">
                      {member.displayName || member.email}
                    </Text>
                    <Text className="ml-auto font-sans text-[11.5px] text-neutral-500">
                      {strings.note.owner}
                    </Text>
                  </View>
                );
              }
              const existing = rows.find(
                (s) => s.subject === ShareSubject.MEMBER && s.memberUserId === member.userId,
              );
              return (
                <ShareRow
                  key={member.userId}
                  label={member.displayName || member.email}
                  avatarName={member.displayName || member.email}
                  permission={existing?.permission}
                  disabled={busy}
                  onSet={(permission) => grant(ShareSubject.MEMBER, member.userId, permission)}
                  onRemove={existing ? () => revoke(existing) : undefined}
                />
              );
            })}
          </View>
        </ScrollView>

        <Text className="font-sans text-[11.5px] text-neutral-600">{strings.share.inviteHint}</Text>
        {shares.data && shares.data.length > 0 ? null : (
          <Text className="font-sans text-[11.5px] text-neutral-600">{strings.share.notShared}</Text>
        )}
      </View>
    </Modal>
  );
}

/** The person (or the family) plus a two-value permission control and a revoke. */
function ShareRow({
  label,
  avatarName,
  icon,
  permission,
  disabled,
  onSet,
  onRemove,
}: {
  label: string;
  avatarName?: string;
  icon?: "users-three";
  permission: SharePermission | undefined;
  disabled: boolean;
  onSet: (permission: SharePermission) => void;
  onRemove?: () => void;
}) {
  return (
    <View className="flex-row items-center gap-[9px] py-[7px]">
      {avatarName ? (
        <Avatar name={avatarName} size={24} />
      ) : (
        <View className="h-[24px] w-[24px] items-center justify-center rounded-full bg-accent-800">
          <Icon name={icon ?? "users-three"} size={13} color={nocturne.accent[200]} />
        </View>
      )}
      <Text className="shrink font-sans text-[13.5px] text-fg" numberOfLines={1}>
        {label}
      </Text>

      <View className="ml-auto flex-row items-center gap-[6px]">
        <PermissionToggle
          label={strings.share.view}
          selected={permission === SharePermission.VIEW}
          disabled={disabled}
          onPress={() => onSet(SharePermission.VIEW)}
        />
        <PermissionToggle
          label={strings.share.edit}
          selected={permission === SharePermission.EDIT}
          disabled={disabled}
          onPress={() => onSet(SharePermission.EDIT)}
        />
        {onRemove ? (
          <IconButton
            icon="x"
            label={strings.share.removeShare}
            onPress={onRemove}
            size={14}
            color={nocturne.neutral[500]}
            disabled={disabled}
          />
        ) : null}
      </View>
    </View>
  );
}

function PermissionToggle({
  label,
  selected,
  disabled,
  onPress,
}: {
  label: string;
  selected: boolean;
  disabled: boolean;
  onPress: () => void;
}) {
  return (
    <Pressable
      accessibilityRole="radio"
      accessibilityLabel={label}
      accessibilityState={{ selected, disabled }}
      disabled={disabled}
      onPress={onPress}
      className={`rounded-md border px-[9px] py-[4px] ${
        selected ? "border-accent bg-accent-900" : "border-neutral-800"
      }`}
    >
      <Text className={`font-med text-[12px] ${selected ? "text-accent-300" : "text-neutral-500"}`}>
        {label}
      </Text>
    </Pressable>
  );
}

/**
 * The overlay the design dims the app with behind a sheet. Drawn as the ground colour at
 * opacity rather than as a fourth hard-coded rgba: the ground is a token, and the scrim is
 * "the ground, mostly opaque".
 */
const SCRIM = { backgroundColor: nocturne.bg, opacity: 0.62 } as const;
