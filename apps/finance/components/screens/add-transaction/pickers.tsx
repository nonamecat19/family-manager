import type { Account, GroupNode, Member } from "@fm/api";
import { Avatar, Icon, Kicker } from "@fm/ui";
import { View } from "react-native";

import {
  dayHeading,
  IconCircle,
  organic,
  Row,
  ScrollSheet,
  tintFor,
  type Translate,
} from "@/components/kit";

export interface MemberSheetProps {
  visible: boolean;
  onClose: () => void;
  title: string;
  members: readonly Member[];
  selectedId: string;
  onSelect: (memberId: string) => void;
}

export function MemberSheet({ visible, onClose, title, members, selectedId, onSelect }: MemberSheetProps) {
  return (
    <ScrollSheet visible={visible} onClose={onClose} title={title} scroll>
      {members.map((member, index) => (
        <Row
          key={member.userId}
          title={member.displayName}
          leading={<Avatar name={member.displayName} index={index} size={34} />}
          trailing={
            member.userId === selectedId ? (
              <Icon name="check" size={18} color={organic.accent.DEFAULT} />
            ) : undefined
          }
          onPress={() => onSelect(member.userId)}
          divider={index < members.length - 1}
        />
      ))}
    </ScrollSheet>
  );
}

export interface AccountSheetProps {
  visible: boolean;
  onClose: () => void;
  title: string;
  shared: readonly Account[];
  privateOwn: readonly Account[];
  sharedLabel: string;
  privateLabel: string;
  selectedId: string;
  onSelect: (accountId: string) => void;
}

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
    <ScrollSheet visible={visible} onClose={onClose} title={title} scroll>
      {shared.length > 0 ? (
        <View className="pb-1.4">
          <Kicker className="mb-1.4">{sharedLabel}</Kicker>
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
        <View className="pt-[8.4px]">
          <Kicker className="mb-1.4">{privateLabel}</Kicker>
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
    </ScrollSheet>
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

export function CategorySheet({
  visible,
  onClose,
  title,
  groups,
  selectedCategoryId,
  onSelect,
}: CategorySheetProps) {
  return (
    <ScrollSheet visible={visible} onClose={onClose} title={title} scroll>
      {groups.map((node) => (
        <View key={node.group?.id ?? ""} className="pb-[8.4px]">
          <Kicker className="mb-1.4">{node.group?.name ?? ""}</Kicker>
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
    </ScrollSheet>
  );
}

export interface DateSheetProps {
  visible: boolean;
  onClose: () => void;
  title: string;
  days: readonly string[];
  selected: string;
  onSelect: (iso: string) => void;
  t: Translate;
}

export function DateSheet({ visible, onClose, title, days, selected, onSelect, t }: DateSheetProps) {
  return (
    <ScrollSheet visible={visible} onClose={onClose} title={title} scroll>
      {days.map((iso, index) => (
        <Row
          key={iso}
          title={dayHeading(t, iso)}
          onPress={() => onSelect(iso)}
          chevron={iso === selected}
          divider={index < days.length - 1}
        />
      ))}
    </ScrollSheet>
  );
}
