import { useState } from "react";
import { Text, View } from "react-native";

import { ErrorText } from "./chrome.tsx";
import { OutlineButton, PrimaryButton } from "./controls.tsx";
import { Field } from "./primitives.tsx";
import {
  SettingsChoiceRow,
  SettingsGroup,
  SettingsLinkRow,
  SettingsSection,
  SettingsToggleRow,
} from "./settings.tsx";
import { useTheme } from "./theme.tsx";

export interface LanguageOption<L extends string> {
  value: L;
  label: string;
}

export function LanguageSection<L extends string>({
  title,
  hint,
  options,
  value,
  onChange,
}: {
  title: string;
  hint?: string;
  options: readonly LanguageOption<L>[];
  value: L;
  onChange: (next: L) => void;
}) {
  return (
    <SettingsSection title={title} hint={hint}>
      <SettingsGroup className="py-[4px]">
        {options.map((option, i) => (
          <SettingsChoiceRow
            key={option.value}
            label={option.label}
            selected={option.value === value}
            divider={i < options.length - 1}
            onPress={() => onChange(option.value)}
          />
        ))}
      </SettingsGroup>
    </SettingsSection>
  );
}

export interface TelegramSectionStrings {
  title: string;
  hint: string;
  waiting: string;
  connected: string;
  notConnected: string;
  connect: string;
  disconnect: string;
}

export interface TelegramLinkState {
  available: boolean;
  awaiting: boolean;
  identity: unknown;
  connecting: boolean;
  loading: boolean;
  unlinking: boolean;
  connect: () => Promise<unknown>;
  unlink: () => Promise<unknown>;
}

export function TelegramSection({
  strings,
  telegram,
  error,
}: {
  strings: TelegramSectionStrings;
  telegram: TelegramLinkState;
  error?: string | null;
}) {
  const t = useTheme();
  if (!telegram.available) return null;
  return (
    <SettingsSection title={strings.title} hint={telegram.awaiting ? strings.waiting : strings.hint}>
      <SettingsGroup className="mb-[12px] py-[14px]">
        <Text className="text-[15.5px] text-fg" style={{ fontFamily: t.fonts?.bold }}>
          {telegram.identity ? strings.connected : strings.notConnected}
        </Text>
      </SettingsGroup>
      {error ? (
        <View className="mb-[12px]">
          <ErrorText>{error}</ErrorText>
        </View>
      ) : null}
      {telegram.identity ? (
        <PrimaryButton
          title={strings.disconnect}
          disabled={telegram.unlinking}
          onPress={() => void telegram.unlink().catch(() => undefined)}
        />
      ) : (
        <PrimaryButton
          title={strings.connect}
          disabled={telegram.connecting || telegram.loading}
          onPress={() => void telegram.connect().catch(() => undefined)}
        />
      )}
    </SettingsSection>
  );
}

export interface NotificationTopic {
  key: string;
  domain: string;
}

export function NotificationsSection({
  title,
  failedText,
  isError,
  topics,
  muted,
  onChange,
}: {
  title: string;
  failedText: string;
  isError: boolean;
  topics: readonly NotificationTopic[];
  muted: readonly string[];
  onChange: (nextMuted: string[]) => void;
}) {
  const t = useTheme();
  const domains = [...new Set(topics.map((topic) => topic.domain))];
  const toggle = (key: string, muteIt: boolean) => {
    const next = new Set(muted);
    if (muteIt) next.add(key);
    else next.delete(key);
    onChange([...next]);
  };

  return (
    <SettingsSection title={title}>
      {isError ? (
        <Text className="text-[13px] text-neutral-600" style={{ fontFamily: t.fonts?.body }}>
          {failedText}
        </Text>
      ) : (
        <SettingsGroup className="py-[4px]">
          {domains.map((domain) => (
            <View key={domain}>
              <SettingsToggleRow
                label={domain}
                value={!muted.includes(domain)}
                onValueChange={(next) => toggle(domain, !next)}
              />
              {topics
                .filter((topic) => topic.domain === domain)
                .map((topic) => (
                  <SettingsToggleRow
                    key={topic.key}
                    label={topic.key}
                    indent={16}
                    value={!muted.includes(domain) && !muted.includes(topic.key)}
                    disabled={muted.includes(domain)}
                    onValueChange={(next) => toggle(topic.key, !next)}
                  />
                ))}
            </View>
          ))}
        </SettingsGroup>
      )}
    </SettingsSection>
  );
}

export function LinkSection({ title, label, onPress }: { title: string; label: string; onPress: () => void }) {
  return (
    <SettingsSection title={title}>
      <SettingsLinkRow label={label} onPress={onPress} />
    </SettingsSection>
  );
}

export function normalizeUserCode(raw: string): string {
  const letters = raw.toUpperCase().replace(/[^A-Z]/g, "").slice(0, 8);
  return letters.length > 4 ? `${letters.slice(0, 4)}-${letters.slice(4)}` : letters;
}

export interface ApproveDeviceStrings {
  body: string;
  placeholder: string;
  approve: string;
  deny: string;
  approved: string;
  denied: string;
}

export function ApproveDeviceForm({
  strings,
  busy,
  error,
  onApprove,
  onDeny,
}: {
  strings: ApproveDeviceStrings;
  busy: boolean;
  error: string | null;
  onApprove: (code: string, done: () => void) => void;
  onDeny: (code: string, done: () => void) => void;
}) {
  const t = useTheme();
  const [code, setCode] = useState("");
  const [decided, setDecided] = useState<"approved" | "denied" | null>(null);
  const canSubmit = code.length === 9 && !busy;

  return (
    <>
      <Text className="text-[14px] leading-[21px] text-neutral-600" style={{ fontFamily: t.fonts?.body }}>
        {strings.body}
      </Text>
      {decided ? (
        <Text className="text-[15.5px] text-fg" style={{ fontFamily: t.fonts?.bold }}>
          {decided === "approved" ? strings.approved : strings.denied}
        </Text>
      ) : (
        <>
          <Field
            label={strings.placeholder}
            value={code}
            onChangeText={(value) => setCode(normalizeUserCode(value))}
            placeholder={strings.placeholder}
            autoCapitalize="characters"
          />
          {error ? <ErrorText>{error}</ErrorText> : null}
          <View className="gap-[12px]">
            <PrimaryButton
              title={strings.approve}
              disabled={!canSubmit}
              onPress={() => onApprove(code, () => setDecided("approved"))}
            />
            <OutlineButton
              title={strings.deny}
              onPress={() => {
                if (!canSubmit) return;
                onDeny(code, () => setDecided("denied"));
              }}
            />
          </View>
        </>
      )}
    </>
  );
}
