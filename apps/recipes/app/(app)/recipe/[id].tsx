import {
  useAddComment,
  useComments,
  useDeleteRecipe,
  useRateRecipe,
  useRecipe,
  useRecipeCategories,
  useRecipeSubcategories,
  useToggleFavorite,
} from "@fm/api";
import { useLocalSearchParams, useRouter } from "expo-router";
import { useState } from "react";
import {
  Alert,
  Image,
  KeyboardAvoidingView,
  Platform,
  Pressable,
  ScrollView,
  Text,
  TextInput,
  View,
} from "react-native";

import { useBasket } from "../../../components/basket.tsx";
import { ClockIcon, HeartIcon, Icon, StarIcon } from "../../../components/organic/icons.tsx";
import { formatDuration } from "../../../components/organic/format.ts";
import { scaleAmount } from "../../../components/organic/scale.ts";
import { initialOf, organic, tintFor } from "../../../components/organic/tokens.ts";
import {
  Display,
  Kicker,
  NutritionStrip,
  OutlineButton,
  PrimaryButton,
  RoundButton,
  Screen,
  SegTabs,
  StarPicker,
  Stepper,
  Tag,
} from "../../../components/organic/ui.tsx";

const TABS = ["Ingredients", "Steps", "Notes"] as const;
type Tab = (typeof TABS)[number];

export default function RecipeDetailScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const router = useRouter();
  const recipe = useRecipe(id);
  const comments = useComments(id);
  const categories = useRecipeCategories();
  const subcategories = useRecipeSubcategories(recipe.data?.categoryId ?? "");
  const addComment = useAddComment();
  const toggleFavorite = useToggleFavorite();
  const deleteRecipe = useDeleteRecipe();
  const rateRecipe = useRateRecipe();
  const basket = useBasket();

  const [tab, setTab] = useState<Tab>("Ingredients");
  const [batch, setBatch] = useState(1);
  const [commentBody, setCommentBody] = useState("");

  if (recipe.isPending) return <Loading />;
  if (recipe.isError) {
    return (
      <Screen>
        <View className="flex-1 justify-center gap-[16px] px-[22px]">
          <Display size={26}>This one got away</Display>
          <Text className="font-fig text-[15px] text-neutral-600">{recipe.error.message}</Text>
          <PrimaryButton title="Try again" onPress={() => void recipe.refetch()} />
        </View>
      </Screen>
    );
  }

  const r = recipe.data;
  if (!r) return null;

  const tint = tintFor(r.categoryId);
  const totalSeconds = r.prepSeconds + r.cookSeconds;
  const categoryName = categories.data?.find((c) => c.id === r.categoryId)?.name;
  const subcategoryName = subcategories.data?.find((s) => s.id === r.subcategoryId)?.name;
  const inBasket = basket.items.some((i) => i.recipeId === r.id);

  const handleDelete = () => {
    Alert.alert("Delete recipe?", "This cannot be undone.", [
      { text: "Cancel", style: "cancel" },
      {
        text: "Delete",
        style: "destructive",
        onPress: () => deleteRecipe.mutate(r.id, { onSuccess: () => router.replace("/(app)") }),
      },
    ]);
  };

  return (
    <Screen edges={["bottom"]}>
      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} className="flex-1">
        <ScrollView showsVerticalScrollIndicator={false} contentContainerClassName="pb-[28px]">
          {/* Hero — the dish sits in a tinted well that runs under the status bar and curves
              away from the content below it. */}
          <View
            className="rounded-b-3xl px-[22px] pb-[26px] pt-[56px]"
            style={{ backgroundColor: tint.bg }}
          >
            <View className="flex-row justify-between">
              <RoundButton icon="back" label="Back" onPress={() => router.back()} tone="translucent" />
              <View className="flex-row gap-[10px]">
                <RoundButton
                  label="Edit recipe"
                  onPress={() => router.push(`/(app)/recipe-edit/${r.id}`)}
                  tone="translucent"
                >
                  <Icon name="pencil" size={19} color={organic.accent[700]} />
                </RoundButton>
                <RoundButton
                  label={r.favoriteCount > 0 ? "Remove from favourites" : "Add to favourites"}
                  onPress={() => toggleFavorite.mutate(r.id)}
                  tone="translucent"
                >
                  <HeartIcon filled={r.favoriteCount > 0} />
                </RoundButton>
              </View>
            </View>
            <View className="mb-[2px] mt-[6px] items-center">
              {r.imageUrl !== "" ? (
                // "contain" on the tint, not "cover": a background-removed PNG should read
                // as the cut-out it is instead of being cropped into a circle.
                <Image
                  source={{ uri: r.imageUrl }}
                  className="h-[186px] w-[186px]"
                  resizeMode="contain"
                />
              ) : (
                <View className="h-[186px] w-[186px] items-center justify-center rounded-full bg-neutral-100/50">
                  <Text className="font-cap text-[64px]" style={{ color: tint.fg }}>
                    {initialOf(r.title)}
                  </Text>
                </View>
              )}
            </View>
          </View>

          <View className="gap-[18px] px-[22px] pb-[24px] pt-[22px]">
            <View>
              {(categoryName ?? subcategoryName) !== undefined && (
                <View className="mb-[10px] flex-row gap-[7px]">
                  {categoryName !== undefined && <Tag label={categoryName} />}
                  {subcategoryName !== undefined && <Tag label={subcategoryName} tone="accent2" />}
                </View>
              )}
              <Display size={29}>{r.title}</Display>
              <View className="mt-[11px] flex-row items-center gap-[14px]">
                {totalSeconds > 0 && (
                  <View className="flex-row items-center gap-[5px]">
                    <ClockIcon />
                    <Text className="font-fig-bold text-[13.5px] text-neutral-700">
                      {formatDuration(totalSeconds)}
                    </Text>
                  </View>
                )}
                <View className="flex-row items-center gap-[5px]">
                  <StarIcon size={15} />
                  <Text className="font-fig-bold text-[13.5px] text-accent-700">
                    {r.rating > 0 ? `${r.rating}.0` : "Unrated"}
                    {r.favoriteCount > 0 ? ` · ♥ ${r.favoriteCount}` : ""}
                  </Text>
                </View>
              </View>
              {r.description !== "" && (
                <Text className="mt-[10px] font-fig text-[15px] leading-[22px] text-neutral-700">
                  {r.description}
                </Text>
              )}
              {/* Per serving, not per recipe — so the figures do not move when the ingredient
                  stepper scales the list. */}
              <NutritionStrip
                className="mt-[14px]"
                kcal={r.nutrition?.kcal ?? 0}
                proteinG={r.nutrition?.proteinG ?? 0}
                fatG={r.nutrition?.fatG ?? 0}
                carbsG={r.nutrition?.carbsG ?? 0}
              />
            </View>

            <SegTabs options={TABS} value={tab} onChange={setTab} />

            {tab === "Ingredients" && (
              <View>
                <View className="mb-[8px] flex-row items-center justify-between">
                  <Text className="font-fig-bold text-[13px] text-neutral-600">Batches</Text>
                  <Stepper value={batch} onChange={setBatch} label="batches" max={6} />
                </View>
                {r.ingredients.length === 0 ? (
                  <Text className="font-fig text-[15px] text-neutral-600">
                    No ingredients written down yet.
                  </Text>
                ) : (
                  r.ingredients.map((ing, i) => (
                    <View
                      key={`${ing.name}-${i}`}
                      className="flex-row items-baseline justify-between gap-[12px] border-b border-divider py-[11px]"
                    >
                      <Text className="font-fig-semi text-[15.5px] text-fg">{ing.name}</Text>
                      <Text className="font-fig-x text-[14px] text-accent-700">
                        {`${scaleAmount(ing.amount, batch)} ${ing.unit}`.trim()}
                      </Text>
                    </View>
                  ))
                )}
              </View>
            )}

            {tab === "Steps" && (
              <View className="gap-[14px]">
                {r.steps.length === 0 ? (
                  <Text className="font-fig text-[15px] text-neutral-600">No steps written down yet.</Text>
                ) : (
                  r.steps.map((step, i) => (
                    <View key={i} className="flex-row gap-[14px]">
                      <View className="h-[30px] w-[30px] flex-none items-center justify-center rounded-full bg-accent2-300">
                        <Text className="font-cap text-[14px] text-accent2-900">{step.position}</Text>
                      </View>
                      <View className="flex-1 pt-[3px]">
                        <Text className="font-fig text-[15.5px] leading-[22px] text-fg">
                          {step.instruction}
                        </Text>
                        {step.durationSeconds > 0 && (
                          <Text className="mt-[4px] font-fig-bold text-[12.5px] text-neutral-600">
                            {formatDuration(step.durationSeconds)}
                          </Text>
                        )}
                      </View>
                    </View>
                  ))
                )}
              </View>
            )}

            {tab === "Notes" && (
              <View className="gap-[12px]">
                {r.notes !== "" && (
                  <View className="rounded-2xl bg-accent2-100 px-[18px] py-[16px]">
                    <Kicker className="text-accent2-700">The cook</Kicker>
                    <Text className="mt-[7px] font-fig text-[15px] leading-[22px] text-fg">{r.notes}</Text>
                  </View>
                )}
                {(comments.data ?? []).map((c) => (
                  <View key={c.id} className="rounded-2xl bg-accent2-100 px-[18px] py-[16px]">
                    <Kicker className="text-accent2-700">
                      {c.createdAt ? new Date(Number(c.createdAt.seconds) * 1000).toLocaleDateString() : "Family"}
                    </Kicker>
                    <Text className="mt-[7px] font-fig text-[15px] leading-[22px] text-fg">{c.body}</Text>
                  </View>
                ))}
                {r.notes === "" && (comments.data ?? []).length === 0 && (
                  <Text className="font-fig text-[15px] text-neutral-600">
                    Nothing written in the margin yet.
                  </Text>
                )}

                <View className="gap-[10px] rounded-2xl bg-neutral-100 px-[16px] py-[14px]">
                  <Kicker>Add a note</Kicker>
                  <TextInput
                    accessibilityLabel="Add a comment"
                    value={commentBody}
                    onChangeText={setCommentBody}
                    multiline
                    placeholder="What did you change?"
                    placeholderTextColor={organic.neutral[500]}
                    className="min-h-[54px] font-fig text-[15px] text-fg"
                  />
                  <PrimaryButton
                    title="Post"
                    disabled={commentBody.trim() === "" || addComment.isPending}
                    onPress={() =>
                      addComment.mutate(
                        { recipeId: r.id, body: commentBody.trim() },
                        { onSuccess: () => setCommentBody("") },
                      )
                    }
                  />
                </View>

                <View className="gap-[8px] pt-[4px]">
                  <Kicker>Your rating</Kicker>
                  <StarPicker
                    rating={r.rating}
                    onChange={(n) => rateRecipe.mutate({ recipeId: r.id, rating: n })}
                  />
                </View>
              </View>
            )}

            <View className="mt-[2px] flex-row gap-[10px]">
              <PrimaryButton
                title={inBasket ? "In the plan" : "Add to plan"}
                onPress={() => {
                  basket.add(r.id);
                  router.push("/(app)/meal-plan");
                }}
                className="flex-1"
              />
              {r.steps.length > 0 && (
                <OutlineButton title="Cook" onPress={() => router.push(`/(app)/cook/${r.id}`)} />
              )}
            </View>

            <Pressable
              accessibilityRole="button"
              accessibilityLabel="Delete recipe"
              onPress={handleDelete}
              className="items-center pt-[6px]"
            >
              <Text className="font-fig-semi text-[13.5px]" style={{ color: organic.danger }}>
                Delete recipe
              </Text>
            </Pressable>
          </View>
        </ScrollView>
      </KeyboardAvoidingView>
    </Screen>
  );
}

function Loading() {
  return (
    <Screen>
      <View className="flex-1 items-center justify-center">
        <Text className="font-fig-semi text-[14px] text-neutral-600">Fetching the recipe…</Text>
      </View>
    </Screen>
  );
}
