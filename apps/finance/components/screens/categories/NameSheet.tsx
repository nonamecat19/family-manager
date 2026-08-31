import { useEffect, useState } from "react";
import { View } from "react-native";

import { Button, Field, Sheet } from "@/components/nocturne";

export interface NameSheetProps {
  visible: boolean;
  /** "Нова група" / "Додати" — the sheet's own heading, also the field's label. */
  title: string;
  saveLabel: string;
  cancelLabel: string;
  submitting: boolean;
  /** A failed mutation, already phrased by the screen. */
  error?: string | null;
  onClose: () => void;
  onSubmit: (name: string) => void;
}

/**
 * Creating a group or a category is one text field, so both use this sheet rather than a
 * route. The name lives here: the screen only learns it on submit, which keeps the screen
 * free of a draft-state branch per sheet.
 */
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

  // Every opening starts empty — a sheet that reopens holding the previous name is how a
  // household ends up with two groups called the same thing.
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
