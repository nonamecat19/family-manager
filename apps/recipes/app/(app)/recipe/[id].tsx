import { toDisplayError, useAddComment, useComments, useDeleteRecipe, useRateRecipe, useRecipe, useRecipeCategories, useRecipeSubcategories, useToggleFavorite } from "@fm/api";
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
import { ClockIcon, HeartIcon, Icon, StarIcon, initialOf, organic, tintFor, Display, Kicker, NutritionStrip, OutlineButton, PrimaryButton, RoundButton, Screen, SegTabs, StarPicker, Stepper, Tag, Loading } from "@fm/ui";

import { useBasket } from "../../../components/basket.tsx";
import { useI18n } from "../../../components/i18n/index.tsx";
import { formatDuration } from "../../../components/organic/format.ts";
import { scaleAmount } from "../../../components/organic/scale.ts";

const TABS = ["Ingredients", "Steps", "Notes"] as const;
type Tab = (typeof TABS)[number];

export default function RecipeDetailScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const router = useRouter();
  const { t, locale } = useI18n();
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

  if (recipe.isPending) return <Fetching />;
  if (recipe.isError) {
    const shown = toDisplayError(recipe.error, t("common.loadFailed"));
    return (
      <Screen>
        <View className="flex-1 justify-center gap-lg px-5.5">
          <Display size={26}>{t("recipeDetail.gotAway")}</Display>
          <Text className="font-fig text-15 text-neutral-600">{shown.message}</Text>
          {shown.reference ? (
            <Text className="font-fig text-13 text-neutral-500">
              {t("common.errorReference", { ref: shown.reference })}
            </Text>
          ) : null}
          <PrimaryButton title={t("common.tryAgain")} onPress={() => void recipe.refetch()} />
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
    Alert.alert(t("recipeDetail.deleteRecipeTitle"), t("recipeDetail.deleteRecipeBody"), [
      { text: t("common.cancel"), style: "cancel" },
      {
        text: t("common.delete"),
        style: "destructive",
        onPress: () => deleteRecipe.mutate(r.id, { onSuccess: () => router.replace("/(app)") }),
      },
    ]);
  };

  return (
    <Screen edges={["bottom"]}>
      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} className="flex-1">
        <ScrollView showsVerticalScrollIndicator={false} contentContainerClassName="pb-7">
          <View
            className="rounded-b-3xl px-5.5 pb-lg pt-14"
            style={{ backgroundColor: tint.bg }}
          >
            <View className="flex-row justify-between">
              <RoundButton icon="back" label={t("common.back")} onPress={() => router.back()} tone="translucent" />
              <View className="flex-row gap-2.5">
                <RoundButton
                  label={t("recipeDetail.editRecipe")}
                  onPress={() => router.push(`/(app)/recipe-edit/${r.id}`)}
                  tone="translucent"
                >
                  <Icon name="pencil" size={19} color={organic.accent[700]} />
                </RoundButton>
                <RoundButton
                  label={r.favoriteCount > 0 ? t("recipeDetail.removeFromFavourites") : t("recipeDetail.addToFavourites")}
                  onPress={() => toggleFavorite.mutate(r.id)}
                  tone="translucent"
                >
                  <HeartIcon filled={r.favoriteCount > 0} />
                </RoundButton>
              </View>
            </View>
            <View className="mb-0.5 mt-0.5 items-center">
              {r.imageUrl !== "" ? (
                <Image
                  source={{ uri: r.imageUrl }}
                  className="h-[244px] w-[244px]"
                  resizeMode="contain"
                />
              ) : (
                <View className="h-[244px] w-[244px] items-center justify-center rounded-full bg-neutral-100/50">
                  <Text className="font-cap text-80" style={{ color: tint.fg }}>
                    {initialOf(r.title)}
                  </Text>
                </View>
              )}
            </View>
          </View>

          <View className="gap-4.5 px-5.5 pb-xl pt-5.5">
            <View>
              {(categoryName ?? subcategoryName) !== undefined && (
                <View className="mb-2.5 flex-row gap-1.75">
                  {categoryName !== undefined && <Tag label={categoryName} />}
                  {subcategoryName !== undefined && <Tag label={subcategoryName} tone="accent2" />}
                </View>
              )}
              <Display size={29}>{r.title}</Display>
              <View className="mt-2.75 flex-row items-center gap-3.5">
                {totalSeconds > 0 && (
                  <View className="flex-row items-center gap-1.25">
                    <ClockIcon />
                    <Text className="font-fig-bold text-13.5 text-neutral-700">
                      {formatDuration(totalSeconds, t)}
                    </Text>
                  </View>
                )}
                <View className="flex-row items-center gap-1.25">
                  <StarIcon size={15} />
                  <Text className="font-fig-bold text-13.5 text-accent-700">
                    {r.rating > 0 ? `${r.rating}.0` : t("recipeDetail.unrated")}
                    {r.favoriteCount > 0 ? ` · ♥ ${r.favoriteCount}` : ""}
                  </Text>
                </View>
              </View>
              {r.description !== "" && (
                <Text className="mt-2.5 font-fig text-15 leading-[22px] text-neutral-700">
                  {r.description}
                </Text>
              )}
              <NutritionStrip
                className="mt-3.5"
                kcal={r.nutrition?.kcal ?? 0}
                proteinG={r.nutrition?.proteinG ?? 0}
                fatG={r.nutrition?.fatG ?? 0}
                carbsG={r.nutrition?.carbsG ?? 0}
              />
            </View>

            <SegTabs
              options={TABS}
              value={tab}
              onChange={setTab}
              labels={{
                Ingredients: t("recipeDetail.tabIngredients"),
                Steps: t("recipeDetail.tabSteps"),
                Notes: t("recipeDetail.tabNotes"),
              }}
            />

            {tab === "Ingredients" && (
              <View>
                <View className="mb-sm flex-row items-center justify-between">
                  <Text className="font-fig-bold text-13 text-neutral-600">{t("recipeDetail.batches")}</Text>
                  <Stepper value={batch} onChange={setBatch} label={t("recipeDetail.batchesLabel")} max={6} />
                </View>
                {r.ingredients.length === 0 ? (
                  <Text className="font-fig text-15 text-neutral-600">
                    {t("recipeDetail.noIngredientsYet")}
                  </Text>
                ) : (
                  r.ingredients.map((ing, i) => (
                    <View
                      key={`${ing.name}-${i}`}
                      className="flex-row items-baseline justify-between gap-3 border-b border-divider py-2.75"
                    >
                      <Text className="font-fig-semi text-15.5 text-fg">{ing.name}</Text>
                      <Text className="font-fig-x text-14 text-accent-700">
                        {`${scaleAmount(ing.amount, batch)} ${ing.unit}`.trim()}
                      </Text>
                    </View>
                  ))
                )}
              </View>
            )}

            {tab === "Steps" && (
              <View className="gap-3.5">
                {r.steps.length === 0 ? (
                  <Text className="font-fig text-15 text-neutral-600">{t("recipeDetail.noStepsYet")}</Text>
                ) : (
                  r.steps.map((step, i) => (
                    <View key={i} className="flex-row gap-3.5">
                      <View className="h-[30px] w-[30px] flex-none items-center justify-center rounded-full bg-accent2-300">
                        <Text className="font-cap text-14 text-accent2-900">{step.position}</Text>
                      </View>
                      <View className="flex-1 pt-0.75">
                        <Text className="font-fig text-15.5 leading-[22px] text-fg">
                          {step.instruction}
                        </Text>
                        {step.durationSeconds > 0 && (
                          <Text className="mt-xs font-fig-bold text-12.5 text-neutral-600">
                            {formatDuration(step.durationSeconds, t)}
                          </Text>
                        )}
                      </View>
                    </View>
                  ))
                )}
              </View>
            )}

            {tab === "Notes" && (
              <View className="gap-3">
                {r.notes !== "" && (
                  <View className="rounded-2xl bg-accent2-100 px-4.5 py-lg">
                    <Kicker className="text-accent2-700">{t("recipeDetail.theCook")}</Kicker>
                    <Text className="mt-1.75 font-fig text-15 leading-[22px] text-fg">{r.notes}</Text>
                  </View>
                )}
                {(comments.data ?? []).map((c) => (
                  <View key={c.id} className="rounded-2xl bg-accent2-100 px-4.5 py-lg">
                    <Kicker className="text-accent2-700">
                      {c.createdAt
                        ? new Date(Number(c.createdAt.seconds) * 1000).toLocaleDateString(locale)
                        : t("recipeDetail.family")}
                    </Kicker>
                    <Text className="mt-1.75 font-fig text-15 leading-[22px] text-fg">{c.body}</Text>
                  </View>
                ))}
                {r.notes === "" && (comments.data ?? []).length === 0 && (
                  <Text className="font-fig text-15 text-neutral-600">
                    {t("recipeDetail.nothingInTheMargin")}
                  </Text>
                )}

                <View className="gap-2.5 rounded-2xl bg-neutral-100 px-lg py-3.5">
                  <Kicker>{t("recipeDetail.addANote")}</Kicker>
                  <TextInput
                    accessibilityLabel={t("recipeDetail.addAComment")}
                    value={commentBody}
                    onChangeText={setCommentBody}
                    multiline
                    placeholder={t("recipeDetail.whatDidYouChange")}
                    placeholderTextColor={organic.neutral[500]}
                    className="min-h-[54px] font-fig text-15 text-fg"
                  />
                  <PrimaryButton
                    title={t("recipeDetail.post")}
                    disabled={commentBody.trim() === "" || addComment.isPending}
                    onPress={() =>
                      addComment.mutate(
                        { recipeId: r.id, body: commentBody.trim() },
                        { onSuccess: () => setCommentBody("") },
                      )
                    }
                  />
                </View>

                <View className="gap-sm pt-xs">
                  <Kicker>{t("recipeDetail.yourRating")}</Kicker>
                  <StarPicker
                    rating={r.rating}
                    onChange={(n) => rateRecipe.mutate({ recipeId: r.id, rating: n })}
                  />
                </View>
              </View>
            )}

            <View className="mt-0.5 flex-row gap-2.5">
              <PrimaryButton
                title={inBasket ? t("recipeDetail.inThePlan") : t("recipeDetail.addToPlan")}
                onPress={() => {
                  basket.add(r.id);
                  router.push("/(app)/meal-plan");
                }}
                className="flex-1"
              />
              {r.steps.length > 0 && (
                <OutlineButton title={t("recipeDetail.cook")} onPress={() => router.push(`/(app)/cook/${r.id}`)} />
              )}
            </View>

            <Pressable
              accessibilityRole="button"
              accessibilityLabel={t("recipeDetail.deleteRecipe")}
              onPress={handleDelete}
              className="items-center pt-1.5"
            >
              <Text className="font-fig-semi text-13.5" style={{ color: organic.danger }}>
                {t("recipeDetail.deleteRecipe")}
              </Text>
            </Pressable>
          </View>
        </ScrollView>
      </KeyboardAvoidingView>
    </Screen>
  );
}

function Fetching() {
  const { t } = useI18n();
  return (
    <Screen>
      <Loading label={t("recipeDetail.fetching")} />
    </Screen>
  );
}
