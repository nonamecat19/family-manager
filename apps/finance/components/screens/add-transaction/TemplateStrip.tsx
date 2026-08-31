import { fromWire, type QuickTemplate } from "@fm/api";
import { Text, View } from "react-native";

import { Chip, iconOr, Kicker } from "@/components/nocturne";

export interface TemplateStripProps {
  templates: readonly QuickTemplate[];
  /** The template whose values are currently in the form, if any. */
  selectedId: string;
  /** Tap — write the transaction straight away. */
  onLog: (template: QuickTemplate) => void;
  /** Long press — drop the template's values into the form and let the user edit. */
  onPrefill: (template: QuickTemplate) => void;
  /** The dashed "Новий" chip. */
  onNew: () => void;
  title: string;
  newLabel: string;
  hint: string;
  fallbackCurrency: string;
}

/**
 * The row the whole screen is built around: the household's quick templates, wrapping, with
 * the add chip last and the two-gesture hint under them. Ordering is the service's
 * `sortOrder` — the list arrives ordered and is not re-sorted here.
 */
export function TemplateStrip({
  templates,
  selectedId,
  onLog,
  onPrefill,
  onNew,
  title,
  newLabel,
  hint,
  fallbackCurrency,
}: TemplateStripProps) {
  return (
    <View>
      <Kicker className="mb-n2">{title}</Kicker>
      <View className="flex-row flex-wrap gap-n2">
        {templates.map((template) => (
          <Chip
            key={template.id}
            label={template.label}
            amount={fromWire(template.amount, fallbackCurrency)}
            icon={iconOr(template.icon)}
            selected={template.id === selectedId}
            onPress={() => onLog(template)}
            onLongPress={() => onPrefill(template)}
          />
        ))}
        <Chip label={newLabel} icon="plus" variant="outline" onPress={onNew} />
      </View>
      <Text className="mt-n3 text-[10.5px] leading-[16px] text-neutral-600">{hint}</Text>
    </View>
  );
}
