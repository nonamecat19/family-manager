import { useFamily, useShareNote, useShareNotebook, useShares, useUnshare } from "@fm/api";
import type { Member } from "@fm/sdk/family/v1/family_pb";
import { SharePermission, ShareSubject, type Share } from "@fm/sdk/notes/v1/notes_pb";
import { useCallback, useMemo } from "react";
import { Modal, Pressable, ScrollView, Text, View } from "react-native";

import { strings } from "./i18n/index.ts";
import { Avatar, Divider, Icon, IconButton, organic } from "@fm/ui";


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
  noteId?: string;
  notebookId?: string;
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
        className="max-h-[70%] gap-3.5 bg-surface px-[18px] pb-[24px] pt-[10px]"
        style={{ borderTopLeftRadius: 20, borderTopRightRadius: 20 }}
      >
        <View className="h-[4px] w-[36px] self-center rounded-md bg-neutral-400" />

        <View className="flex-row items-center">
          <Text className="font-fig-med text-[16px] text-fg">
            {notebookId ? strings.share.titleNotebook : strings.share.title}
          </Text>
          <View className="ml-auto">
            <IconButton icon="x" label={strings.common.close} onPress={onClose} size={17} color={organic.neutral[700]} />
          </View>
        </View>

        <Text className="font-fig text-[12.5px] leading-[19px] text-neutral-700">
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
              <Text className="py-[8px] font-fig text-[12.5px] text-neutral-700">
                {strings.share.noMembers}
              </Text>
            ) : null}

            {members.map((member) => {
              if (member.userId === ownerUserId) {
                return (
                  <View key={member.userId} className="flex-row items-center gap-[9px] py-[9px]">
                    <Avatar name={member.displayName || member.email} size={24} />
                    <Text className="font-fig text-[13.5px] text-fg">
                      {member.displayName || member.email}
                    </Text>
                    <Text className="ml-auto font-fig text-[11.5px] text-neutral-700">
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

        <Text className="font-fig text-[11.5px] text-neutral-600">{strings.share.inviteHint}</Text>
        {shares.data && shares.data.length > 0 ? null : (
          <Text className="font-fig text-[11.5px] text-neutral-600">{strings.share.notShared}</Text>
        )}
      </View>
    </Modal>
  );
}

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
        <View className="h-[24px] w-[24px] items-center justify-center rounded-full bg-accent-200">
          <Icon name={icon ?? "users-three"} size={13} color={organic.accent[800]} />
        </View>
      )}
      <Text className="shrink font-fig text-[13.5px] text-fg" numberOfLines={1}>
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
            color={organic.neutral[700]}
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
      className={`rounded-xl border px-[9px] py-[4px] ${
        selected ? "border-accent bg-accent-100" : "border-neutral-300"
      }`}
    >
      <Text className={`font-fig-med text-[12px] ${selected ? "text-accent-800" : "text-neutral-700"}`}>
        {label}
      </Text>
    </Pressable>
  );
}

const SCRIM = { backgroundColor: organic.bg, opacity: 0.62 } as const;
