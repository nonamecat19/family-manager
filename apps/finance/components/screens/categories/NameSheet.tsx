import { useEffect, useState } from "react";
import { View } from "react-native";

import { Button, Field, Sheet } from "@/components/nocturne";

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
    <Sheet visible={visible} onClose={onClose} title={title}>
      <View className="gap-n5 pt-n2">
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
        <View className="flex-row gap-n3">
          <Button title={cancelLabel} variant="ghost" onPress={onClose} className="flex-1" />
          <Button
            title={saveLabel}
            onPress={() => onSubmit(trimmed)}
            disabled={trimmed === "" || submitting}
            className="flex-1"
          />
        </View>
      </View>
    </Sheet>
  );
}
