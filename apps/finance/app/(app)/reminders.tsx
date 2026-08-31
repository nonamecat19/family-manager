import {
  ReminderKind,
  toDisplayError,
  useDeleteReminder,
  useFamily,
  useReminders,
  useUpsertReminder,
  type Reminder,
} from "@fm/api";
import { useRouter } from "expo-router";
import { useState } from "react";
import { Alert, ScrollView, Text, View } from "react-native";

import { useI18n } from "@/components/i18n";
import {
  Button,
  Drawer,
  EmptyState,
  Fab,
  Field,
  IconCircle,
  ListSection,
  Row,
  Screen,
  ScreenHeader,
  SegmentedTabs,
  Sheet,
  useDrawerItems,
  type IconName,
} from "@/components/nocturne";

/**
 * Нагадування — the drawer's fourth row.
 *
 * Three kinds, and the kind is what decides when the app speaks: a budget breach, a due
 * recurring payment, or a plain one the household wrote themselves. A reminder is toggled far
 * more often than it is written, so the switch is on the row and the sheet is for the rest.
 */
export default function RemindersScreen() {
  const { t } = useI18n();
  const router = useRouter();
  const drawerItems = useDrawerItems();

  const [drawerOpen, setDrawerOpen] = useState(false);
  const [creating, setCreating] = useState(false);

  const reminders = useReminders(true);
  const family = useFamily();
  const upsert = useUpsertReminder();
  const remove = useDeleteReminder();

  const list = reminders.data ?? [];

  const confirmRemove = (reminder: Reminder) => {
    Alert.alert(t("reminders.deleteTitle"), t("reminders.deleteBody", { title: reminder.title }), [
      { text: t("common.cancel"), style: "cancel" },
      { text: t("common.delete"), style: "destructive", onPress: () => remove.mutate(reminder.id) },
    ]);
  };

  return (
    <Screen>
      <ScreenHeader
        gradient
        title={t("reminders.title")}
        subtitle={t("reminders.subtitle")}
        leading={{ icon: "list", label: t("nav.menu"), onPress: () => setDrawerOpen(true) }}
      />

      {reminders.isPending ? (
        <View className="flex-1 items-center justify-center">
          <Text className="text-[13px] text-neutral-500">{t("common.loadingEllipsis")}</Text>
        </View>
      ) : reminders.isError ? (
        <View className="flex-1 justify-center gap-n3 px-n5">
          <Text className="text-[13.5px] leading-[21px] text-neutral-500">
            {toDisplayError(reminders.error, t("common.loadFailed")).message}
          </Text>
          <Button title={t("common.tryAgain")} onPress={() => void reminders.refetch()} />
        </View>
      ) : list.length === 0 ? (
        <View className="flex-1 justify-center">
          <EmptyState
            icon="bell"
            title={t("reminders.emptyTitle")}
            body={t("reminders.emptyBody")}
            action={{ label: t("reminders.new"), onPress: () => setCreating(true) }}
          />
        </View>
      ) : (
        <ScrollView className="flex-1 px-n4" contentContainerClassName="pb-[96px] pt-n4">
          <ListSection title={t("reminders.all")}>
            {list.map((reminder, index) => (
              <Row
                key={reminder.id}
                title={reminder.title}
                subtitle={t(kindKey(reminder.kind))}
                leading={<IconCircle icon={kindIcon(reminder.kind)} />}
                trailing={
                  <Text className="text-[12px] text-neutral-500">
                    {reminder.enabled ? t("reminders.on") : t("reminders.off")}
                  </Text>
                }
                // The row is the switch: one tap flips it, which is the only thing anyone
                // does to a reminder day to day.
                onPress={() =>
                  upsert.mutate({
                    reminderId: reminder.id,
                    kind: reminder.kind,
                    title: reminder.title,
                    enabled: !reminder.enabled,
                  })
                }
                onLongPress={() => confirmRemove(reminder)}
                divider={index < list.length - 1}
              />
            ))}
          </ListSection>

          <Text className="mt-n3 px-n2 text-[11px] text-neutral-500">{t("reminders.tapHint")}</Text>
        </ScrollView>
      )}

      <Fab label={t("reminders.new")} onPress={() => setCreating(true)} />

      <NewReminderSheet visible={creating} onClose={() => setCreating(false)} />

      <Drawer
        visible={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        account={{ name: family.data?.family?.name ?? "", email: "" }}
        household={{ name: family.data?.family?.name ?? "" }}
        items={drawerItems}
        activeId="reminders"
        onSelect={(item) => {
          setDrawerOpen(false);
          if (item.href && item.id !== "reminders") router.push(item.href);
        }}
      />
    </Screen>
  );
}

type Kind = "budget" | "recurring" | "custom";

function kindKey(kind: ReminderKind) {
  if (kind === ReminderKind.BUDGET_EXCEEDED) return "reminders.kindBudget" as const;
  if (kind === ReminderKind.RECURRING_DUE) return "reminders.kindRecurring" as const;
  return "reminders.kindCustom" as const;
}

function kindIcon(kind: ReminderKind): IconName {
  if (kind === ReminderKind.BUDGET_EXCEEDED) return "target";
  if (kind === ReminderKind.RECURRING_DUE) return "arrows-clockwise";
  return "bell";
}

function NewReminderSheet({ visible, onClose }: { visible: boolean; onClose: () => void }) {
  const { t } = useI18n();

  const [title, setTitle] = useState("");
  const [kind, setKind] = useState<Kind>("custom");
  const [error, setError] = useState<string | null>(null);

  const upsert = useUpsertReminder();

  const save = async () => {
    setError(null);
    if (title.trim() === "") {
      setError(t("reminders.titleRequired"));
      return;
    }
    try {
      await upsert.mutateAsync({
        title: title.trim(),
        kind:
          kind === "budget"
            ? ReminderKind.BUDGET_EXCEEDED
            : kind === "recurring"
              ? ReminderKind.RECURRING_DUE
              : ReminderKind.CUSTOM,
        enabled: true,
      });
      setTitle("");
      onClose();
    } catch (e) {
      setError(toDisplayError(e, t("reminders.saveError")).message);
    }
  };

  return (
    <Sheet visible={visible} onClose={onClose} title={t("reminders.new")} scroll>
      <View className="gap-n4">
        <Field label={t("reminders.titleField")} value={title} onChangeText={setTitle} />

        <SegmentedTabs<Kind>
          value={kind}
          onChange={setKind}
          options={[
            { value: "custom", label: t("reminders.kindCustom") },
            { value: "budget", label: t("reminders.kindBudget") },
            { value: "recurring", label: t("reminders.kindRecurring") },
          ]}
        />

        {error ? <Text className="text-[12.5px] text-overspend">{error}</Text> : null}

        <Button title={t("common.save")} onPress={() => void save()} disabled={upsert.isPending} />
      </View>
    </Sheet>
  );
}
