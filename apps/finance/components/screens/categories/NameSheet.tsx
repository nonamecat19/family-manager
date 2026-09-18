import { useEffect, useState } from "react";
import { View } from "react-native";
import { Button, Field } from "@fm/ui";
import { ScrollSheet } from "@/components/kit";

export interface NameSheetProps {
  visible: boolean;
  title: string;
  saveLabel: string;
  cancelLabel: string;
  submitting: boolean;
  error?: string | null;
  onClose: () => void;
  onSubmit: (name: string) => void;
}

export function NameSheet({
  visible,
  title,
  saveLabel,
  cancelLabel,
  submitting,
  error,
  onClose,
  onSubmit,
}: NameSheetProps) {
  const [name, setName] = useState("");

  useEffect(() => {
    if (visible) setName("");
  }, [visible]);

  const trimmed = name.trim();

  return (
    <ScrollSheet visible={visible} onClose={onClose} title={title}>
      <View className="gap-4.2 pt-1.4">
        <Field
          label={title}
          value={name}
          onChangeText={setName}
          autoFocus
          autoCapitalize="sentences"
          returnKeyType="done"
          onSubmitEditing={() => trimmed !== "" && !submitting && onSubmit(trimmed)}
          error={error ?? undefined}
        />
        <View className="flex-row gap-2.1">
          <Button title={cancelLabel} tone="quiet" onPress={onClose} />
          <Button
            title={saveLabel}
            onPress={() => onSubmit(trimmed)}
            disabled={trimmed === "" || submitting}
          />
        </View>
      </View>
    </ScrollSheet>
  );
}
