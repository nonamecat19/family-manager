import type { Money } from "@fm/api";
import type { Href } from "expo-router";
import { Modal, Pressable, ScrollView, Text, View } from "react-native";

import { useI18n } from "../i18n/index.tsx";
import { Icon, type IconName } from "./icons.tsx";
import { nocturne } from "./tokens.ts";
import { Divider, MemberAvatar, MoneyText } from "./ui.tsx";

export interface DrawerItem {
  id: string;
  icon: IconName;
  label: string;
  /** An expo-router href. The kit does not navigate — `onSelect` gets the whole item and the
   * screen decides. */
  href?: Href;
}

/**
 * The nine destinations screen 11's overlay lists, already translated. A screen calls this
 * and passes the result straight to `Drawer`; it must not retype the list, because the copy
 * lives in `components/i18n` and screens do not edit translations.
 */
export function useDrawerItems(): DrawerItem[] {
  const { t } = useI18n();
  return [
    { id: "home", icon: "squares-four", label: t("nav.home"), href: "/(app)" },
    { id: "accounts", icon: "wallet", label: t("nav.accounts"), href: "/(app)/accounts" },
    { id: "charts", icon: "chart-bar", label: t("nav.charts"), href: "/(app)/charts" },
    { id: "categories", icon: "squares-four", label: t("nav.categories"), href: "/(app)/categories" },
    { id: "household", icon: "users-three", label: t("nav.household"), href: "/(app)/household" },
    { id: "templates", icon: "lightning", label: t("nav.templates"), href: "/(app)/templates" },
    { id: "recurring", icon: "arrows-clockwise", label: t("nav.recurring"), href: "/(app)/recurring" },
    { id: "reminders", icon: "bell", label: t("nav.reminders"), href: "/(app)/reminders" },
    { id: "settings", icon: "gear", label: t("nav.settings"), href: "/(app)/settings" },
  ];
}

export interface DrawerScope {
  id: string;
  label: string;
}

export interface DrawerProps {
  visible: boolean;
  onClose: () => void;
  /** The signed-in person, drawn at the top. */
  account: { name: string; email: string };
  /** The household line under it. */
  household?: { name: string; balance?: Money };
  /** Родина | Сергій | Олена — the same scopes the Home switcher offers, as a compact row. */
  scopes?: readonly DrawerScope[];
  activeScopeId?: string;
  onSelectScope?: (id: string) => void;
  items: readonly DrawerItem[];
  activeId?: string;
  onSelect: (item: DrawerItem) => void;
  /** "Синхронізовано 11:38". */
  footer?: string;
}

/**
 * The navigation overlay from screen 11: a scrim plus a panel over the current screen, not a
 * route of its own. Deliberately not `expo-router`'s Drawer layout — the design shows it over
 * a stack, the app's primary navigation is the Home screen itself, and a drawer navigator
 * would put a second gesture on every screen edge.
 */
export function Drawer({
  visible,
  onClose,
  account,
  household,
  scopes,
  activeScopeId,
  onSelectScope,
  items,
  activeId,
  onSelect,
  footer,
}: DrawerProps) {
  const { t } = useI18n();
  return (
    <Modal visible={visible} transparent animationType="fade" onRequestClose={onClose}>
      <View className="flex-1 flex-row">
        <View className="w-[80%] max-w-[320px] bg-bg pt-[44px]">
          <View className="flex-row items-center gap-n4 px-n5 pb-n5">
            <MemberAvatar name={account.name} size={38} />
            <View className="flex-1">
              <Text className="text-[13.5px] font-medium text-fg" numberOfLines={1}>
                {account.email}
              </Text>
              {household ? (
                <View className="mt-[2px] flex-row items-center gap-n2">
                  <Text className="text-[11.5px] text-neutral-500" numberOfLines={1}>
                    {household.name}
                  </Text>
                  {household.balance ? (
                    <MoneyText value={household.balance} size={11.5} tone="accent" weight="medium" />
                  ) : null}
                </View>
              ) : null}
            </View>
          </View>

          {scopes && scopes.length > 0 ? (
            <ScrollView
              horizontal
              showsHorizontalScrollIndicator={false}
              contentContainerStyle={{ gap: nocturne.space.n2, paddingHorizontal: nocturne.space.n5 }}
              className="pb-n4"
            >
              {scopes.map((scope) => {
                const active = scope.id === activeScopeId;
                return (
                  <Pressable
                    key={scope.id}
                    accessibilityRole="button"
                    accessibilityLabel={scope.label}
                    accessibilityState={{ selected: active }}
                    onPress={() => onSelectScope?.(scope.id)}
                    className={`flex-none rounded-full px-n4 py-[5px] ${active ? "bg-accent-800" : "bg-surface"}`}
                  >
                    <Text
                      className={`text-[11.5px] font-medium ${active ? "text-accent-100" : "text-neutral-400"}`}
                    >
                      {scope.label}
                    </Text>
                  </Pressable>
                );
              })}
            </ScrollView>
          ) : null}

          <Divider />

          <ScrollView className="flex-1 pt-n3">
            {items.map((item) => {
              const active = item.id === activeId;
              return (
                <Pressable
                  key={item.id}
                  accessibilityRole="link"
                  accessibilityLabel={item.label}
                  accessibilityState={{ selected: active }}
                  onPress={() => onSelect(item)}
                  className={`mx-n3 flex-row items-center gap-n4 rounded-md px-n4 py-n4 ${
                    active ? "bg-accent-900" : ""
                  }`}
                >
                  <Icon
                    name={item.icon}
                    size={19}
                    color={active ? nocturne.accent[300] : nocturne.neutral[500]}
                  />
                  <Text className={`text-[14px] ${active ? "font-medium text-fg" : "text-neutral-300"}`}>
                    {item.label}
                  </Text>
                </Pressable>
              );
            })}
          </ScrollView>

          {footer ? (
            <View className="px-n5 py-n4">
              <Divider className="mb-n4" />
              <Text className="text-[11px] text-neutral-600">{footer}</Text>
            </View>
          ) : null}
        </View>

        <Pressable
          accessibilityRole="button"
          accessibilityLabel={t("common.close")}
          onPress={onClose}
          className="flex-1"
          style={{ backgroundColor: nocturne.scrim }}
        />
      </View>
    </Modal>
  );
}
