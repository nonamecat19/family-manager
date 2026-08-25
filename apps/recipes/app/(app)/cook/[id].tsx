import { useRecipe } from "@fm/api";
import { useLocalSearchParams, useRouter } from "expo-router";
import { useEffect, useRef, useState } from "react";
import { Pressable, Text, View } from "react-native";

import { Icon } from "../../../components/organic/icons.tsx";
import { organic } from "../../../components/organic/tokens.ts";
import { Screen } from "../../../components/organic/ui.tsx";

/**
 * Cook mode: one step, filling the screen, on a dark ground so a phone propped against a
 * bowl is readable at arm's length. It is the only surface in the app that inverts — the
 * point is that you are not browsing any more.
 */
export default function CookScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const router = useRouter();
  const recipe = useRecipe(id);

  const [index, setIndex] = useState(0);
  const steps = recipe.data?.steps ?? [];
  const step = steps[index];

  return (
    <View className="flex-1 bg-accent-900">
      <Screen edges={["top", "bottom"]} className="bg-transparent">
        <View className="flex-1 px-[26px] pb-[34px] pt-[16px]">
          <View className="flex-row items-center justify-between">
            <Pressable
              accessibilityRole="button"
              accessibilityLabel="Leave cook mode"
              onPress={() => router.back()}
              className="h-10 w-10 items-center justify-center rounded-full"
              style={{ backgroundColor: "rgba(255,255,255,0.14)" }}
            >
              <Icon name="close" size={20} color={organic.accent[100]} />
            </Pressable>
            <Text className="font-fig-x text-[13px] uppercase tracking-[1.5px] text-accent-100 opacity-65">
              {steps.length > 0 ? `Step ${index + 1} of ${steps.length}` : "Cook"}
            </Text>
            <View className="w-10" />
          </View>

          <View className="mt-[22px] flex-row gap-[5px]">
            {steps.map((s, i) => (
              <View
                key={i}
                className="h-[4px] flex-1 rounded-full"
                style={{
                  backgroundColor: i <= index ? organic.accent.DEFAULT : "rgba(255,255,255,0.22)",
                }}
              />
            ))}
          </View>

          <View className="flex-1 justify-center gap-[26px] py-[30px]">
            <View className="h-[74px] w-[74px] items-center justify-center rounded-full bg-accent">
              <Text className="font-cap text-[30px] text-white">{index + 1}</Text>
            </View>
            <Text className="font-cap text-[30px] leading-[37px] text-accent-100">
              {step?.instruction ?? "This recipe has no steps written down yet."}
            </Text>
            {step && step.durationSeconds > 0 && (
              <StepTimer key={`${index}-${step.durationSeconds}`} seconds={step.durationSeconds} />
            )}
          </View>

          <View className="flex-row gap-[12px]">
            <Pressable
              accessibilityRole="button"
              accessibilityLabel="Previous step"
              disabled={index === 0}
              onPress={() => setIndex((i) => Math.max(0, i - 1))}
              className={`h-[56px] w-[56px] flex-none items-center justify-center rounded-full border-2 ${
                index === 0 ? "opacity-40" : ""
              }`}
              style={{ borderColor: "rgba(255,255,255,0.3)" }}
            >
              <Icon name="back" size={22} color={organic.accent[100]} />
            </Pressable>
            <Pressable
              accessibilityRole="button"
              accessibilityLabel={index >= steps.length - 1 ? "Finish cooking" : "Next step"}
              onPress={() => {
                if (index >= steps.length - 1) router.back();
                else setIndex((i) => i + 1);
              }}
              className="h-[56px] flex-1 items-center justify-center rounded-full bg-accent"
            >
              <Text className="font-fig-x text-[16px] text-white">
                {index >= steps.length - 1 ? "Done" : "Next step"}
              </Text>
            </Pressable>
          </View>
        </View>
      </Screen>
    </View>
  );
}

/**
 * The step's own timer. It starts paused: a step's duration is how long it takes, not a
 * countdown that should begin the moment you happen to swipe onto it.
 */
function StepTimer({ seconds }: { seconds: number }) {
  const [remaining, setRemaining] = useState(seconds);
  const [running, setRunning] = useState(false);
  const tick = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    if (!running) return;
    tick.current = setInterval(() => {
      setRemaining((r) => {
        if (r <= 1) {
          setRunning(false);
          return 0;
        }
        return r - 1;
      });
    }, 1000);
    return () => {
      if (tick.current) clearInterval(tick.current);
    };
  }, [running]);

  const done = remaining === 0;

  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={running ? "Pause timer" : done ? "Reset timer" : "Start timer"}
      onPress={() => {
        if (done) {
          setRemaining(seconds);
          setRunning(false);
        } else {
          setRunning((r) => !r);
        }
      }}
      className="flex-row items-center gap-[10px] self-start rounded-full px-[20px] py-[12px]"
      style={{ backgroundColor: done ? organic.accent.DEFAULT : "rgba(255,255,255,0.1)" }}
    >
      <Icon name="plan" size={18} color={organic.accent[100]} width={2.4} />
      <Text className="font-fig-x text-[16px] text-accent-100">
        {done ? "Time's up — tap to reset" : mmss(remaining)}
      </Text>
      {!done && (
        <Text className="font-fig-semi text-[13px] text-accent-100 opacity-70">
          {running ? "tap to pause" : "tap to start"}
        </Text>
      )}
    </Pressable>
  );
}

function mmss(total: number): string {
  const mins = Math.floor(total / 60);
  const secs = total % 60;
  return `${mins}:${`${secs}`.padStart(2, "0")}`;
}
