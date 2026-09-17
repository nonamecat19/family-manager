import {
  ReminderKind,
  toDisplayError,
  useDeleteReminder,
  useReminders,
  useUpsertReminder,
  type Reminder,
} from "@fm/api";
import { useRouter } from "expo-router";
import { useState } from "react";
import { Alert, Text, View } from "react-native";

import { useI18n } from "@/components/i18n";
import { Fab, IconCircle, ListSection, Row, ScrollSheet, SegmentedTabs } from "@/components/kit";
import { Button, EmptyState, Field, Screen, ScreenHeader, ScrollBody, IconName } from "@fm/ui";

export default function RemindersScreen() {
  const { t } = useI18n();
  const router = useRouter();

  const [creating, setCreating] = useState(false);

  const reminders = useReminders(true);
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
      <ScrollBody>
        <ScreenHeader
          title={t("reminders.title")}
          onBack={() => router.back()}
          backLabel={t("common.back")}
        />

        {reminders.isPending ? (
          <View className="items-center py-7">
            <Text className="text-13 text-neutral-600">{t("common.loadingEllipsis")}</Text>
          </View>
        ) : reminders.isError ? (
          <View className="gap-3 py-4.5">
            <Text className="text-13.5 leading-[21px] text-neutral-600">
              {toDisplayError(reminders.error, t("common.loadFailed")).message}
            </Text>
            <Button title={t("common.tryAgain")} onPress={() => void reminders.refetch()} />
          </View>
        ) : list.length === 0 ? (
          <EmptyState
            title={t("reminders.emptyTitle")}
            body={t("reminders.emptyBody")}
            action={{ label: t("reminders.new"), onPress: () => setCreating(true) }}
          />
        ) : (
          <>
            <ListSection title={t("reminders.all")}>
              {list.map((reminder, index) => (
                <Row
                  key={reminder.id}
                  title={reminder.title}
                  subtitle={t(kindKey(reminder.kind))}
                  leading={<IconCircle icon={kindIcon(reminder.kind)} />}
                  trailing={
                    <Text className="text-12 text-neutral-600">
                      {reminder.enabled ? t("reminders.on") : t("reminders.off")}
                    </Text>
                  }
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

            <Text className="text-11 text-neutral-600">{t("reminders.tapHint")}</Text>
          </>
        )}
      </ScrollBody>

      <Fab label={t("reminders.new")} onPress={() => setCreating(true)} />

      <NewReminderSheet visible={creating} onClose={() => setCreating(false)} />
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
    <ScrollSheet visible={visible} onClose={onClose} title={t("reminders.new")} scroll>
      <View className="gap-2.8">
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

        {error ? <Text className="text-12.5 text-error">{error}</Text> : null}

        <Button title={t("common.save")} onPress={() => void save()} disabled={upsert.isPending} />
      </View>
    </ScrollSheet>
  );
}
