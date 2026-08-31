import type { Account, GroupNode, Member } from "@fm/api";
import { View } from "react-native";

import {
  dayHeading,
  IconCircle,
  Kicker,
  MemberAvatar,
  Row,
  Sheet,
  tintFor,
  type Translate,
} from "@/components/nocturne";

export interface MemberSheetProps {
  visible: boolean;
  onClose: () => void;
  title: string;
  members: readonly Member[];
  selectedId: string;
  onSelect: (memberId: string) => void;
}

/** "Хто" — the household's active members. Pending invitees cannot have spent anything yet. */
export function MemberSheet({ visible, onClose, title, members, selectedId, onSelect }: MemberSheetProps) {
  return (
    <Sheet visible={visible} onClose={onClose} title={title} scroll>
      {members.map((member, index) => (
        <Row
          key={member.userId}
          title={member.displayName}
          leading={<MemberAvatar name={member.displayName} index={index} selected={member.userId === selectedId} />}
          onPress={() => onSelect(member.userId)}
          divider={index < members.length - 1}
        />
      ))}
    </Sheet>
  );
}

export interface AccountSheetProps {
  visible: boolean;
  onClose: () => void;
  title: string;
  /** Accounts the whole household sees. */
  shared: readonly Account[];
  /** The caller's own private accounts — selectable, but labelled as theirs alone. */
  privateOwn: readonly Account[];
  sharedLabel: string;
  privateLabel: string;
  selectedId: string;
  onSelect: (accountId: string) => void;
}

/**
 * "Рахунок". Another member's private accounts are never in this list — the service does not
 * serialise them at all — so nothing here can put a hidden balance in front of the wrong person.
 */
export function AccountSheet({
  visible,
  onClose,
  title,
  shared,
  privateOwn,
  sharedLabel,
  privateLabel,
  selectedId,
  onSelect,
}: AccountSheetProps) {
  return (
    <Sheet visible={visible} onClose={onClose} title={title} scroll>
      {shared.length > 0 ? (
        <View className="pb-n2">
          <Kicker className="mb-n2">{sharedLabel}</Kicker>
          {shared.map((account, index) => (
            <Row
              key={account.id}
              title={account.name}
              leading={<IconCircle icon={account.icon} tint={tintFor(account.colorStep)} size={32} />}
              onPress={() => onSelect(account.id)}
              chevron={account.id === selectedId}
              divider={index < shared.length - 1}
            />
          ))}
        </View>
      ) : null}
      {privateOwn.length > 0 ? (
        <View className="pt-n3">
          <Kicker className="mb-n2">{privateLabel}</Kicker>
          {privateOwn.map((account, index) => (
            <Row
              key={account.id}
              title={account.name}
              leading={<IconCircle icon={account.icon} tint={tintFor(account.colorStep)} size={32} />}
              onPress={() => onSelect(account.id)}
              chevron={account.id === selectedId}
              divider={index < privateOwn.length - 1}
            />
          ))}
        </View>
      ) : null}
    </Sheet>
  );
}

export interface CategorySheetProps {
  visible: boolean;
  onClose: () => void;
  title: string;
  groups: readonly GroupNode[];
  selectedCategoryId: string;
  onSelect: (groupId: string, categoryId: string) => void;
}

/** The full two-level list behind "Ще" and behind the group name beside "Категорія". */
export function CategorySheet({
  visible,
  onClose,
  title,
  groups,
  selectedCategoryId,
  onSelect,
}: CategorySheetProps) {
  return (
    <Sheet visible={visible} onClose={onClose} title={title} scroll>
      {groups.map((node) => (
        <View key={node.group?.id ?? ""} className="pb-n3">
          <Kicker className="mb-n2">{node.group?.name ?? ""}</Kicker>
          {node.categories.map((category, index) => (
            <Row
              key={category.id}
              title={category.name}
              leading={
                <IconCircle
                  icon={category.icon}
                  tint={tintFor(node.group?.colorStep ?? index)}
                  size={32}
                />
              }
              onPress={() => onSelect(node.group?.id ?? "", category.id)}
              chevron={category.id === selectedCategoryId}
              divider={index < node.categories.length - 1}
            />
          ))}
        </View>
      ))}
    </Sheet>
  );
}

export interface DateSheetProps {
  visible: boolean;
  onClose: () => void;
  title: string;
  /** Candidate days, newest first. */
  days: readonly string[];
  selected: string;
  onSelect: (iso: string) => void;
  t: Translate;
}

/**
 * The calendar affordance. A list of recent days rather than a month grid: the app ships no
 * date-picker dependency, and a transaction typed by hand is days old, not months.
 */
export function DateSheet({ visible, onClose, title, days, selected, onSelect, t }: DateSheetProps) {
  return (
    <Sheet visible={visible} onClose={onClose} title={title} scroll>
      {days.map((iso, index) => (
        <Row
          key={iso}
          title={dayHeading(t, iso)}
          onPress={() => onSelect(iso)}
          chevron={iso === selected}
          divider={index < days.length - 1}
        />
      ))}
    </Sheet>
  );
}
