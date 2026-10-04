import { fromWire, type QuickTemplate } from "@fm/api";
import { Text, View } from "react-native";

import { AmountChip } from "@/components/kit";
import { iconOr, Kicker } from "@fm/ui";

export interface TemplateStripProps {
  templates: readonly QuickTemplate[];
  selectedId: string;
  onLog: (template: QuickTemplate) => void;
  onPrefill: (template: QuickTemplate) => void;
  onNew: () => void;
  title: string;
  newLabel: string;
  hint: string;
  fallbackCurrency: string;
}

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
      <Kicker className="mb-[5.6px]">{title}</Kicker>
      <View className="flex-row flex-wrap gap-[5.6px]">
        {templates.map((template) => (
          <AmountChip
            key={template.id}
            label={template.label}
            amount={fromWire(template.amount, fallbackCurrency)}
            icon={iconOr(template.icon)}
            selected={template.id === selectedId}
            onPress={() => onLog(template)}
            onLongPress={() => onPrefill(template)}
          />
        ))}
        <AmountChip label={newLabel} icon="plus" variant="outline" onPress={onNew} />
      </View>
      <Text className="mt-[8.4px] text-[10.5px] leading-[16px] text-neutral-600">{hint}</Text>
    </View>
  );
}
