import { useCreateRecipe, useUpdateRecipe, useRecipe, useRecipeCategories } from "@fm/api";
import { Button, Card, ErrorState, Field, Loading } from "@fm/ui";
import { useLocalSearchParams, useRouter } from "expo-router";
import { useState } from "react";
import { Pressable, ScrollView, Text, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

interface IngredientRow {
  name: string;
  amount: string;
  unit: string;
}

interface StepRow {
  instruction: string;
  durationSeconds: string;
}

export default function RecipeEditScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const router = useRouter();
  const isNew = id === "new";
  const recipe = useRecipe(isNew ? "" : id);
  const categories = useRecipeCategories();
  const createRecipe = useCreateRecipe();
  const updateRecipe = useUpdateRecipe();

  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [servings, setServings] = useState("1");
  const [prepMinutes, setPrepMinutes] = useState("0");
  const [cookMinutes, setCookMinutes] = useState("0");
  const [categoryId, setCategoryId] = useState("");
  const [ingredients, setIngredients] = useState<IngredientRow[]>([{ name: "", amount: "", unit: "" }]);
  const [steps, setSteps] = useState<StepRow[]>([{ instruction: "", durationSeconds: "" }]);

  // Load existing recipe into the form once available.
  if (!isNew && recipe.isPending) return <Loading label="Loading recipe…" />;
  if (!isNew && recipe.isError) return <ErrorState message={recipe.error.message} onRetry={() => void recipe.refetch()} />;
  if (!isNew && recipe.data && title === "" && recipe.data.title !== "") {
    const r = recipe.data;
    setTitle(r.title);
    setDescription(r.description);
    setServings(`${r.servings}`);
    setPrepMinutes(`${Math.floor(r.prepSeconds / 60)}`);
    setCookMinutes(`${Math.floor(r.cookSeconds / 60)}`);
    setCategoryId(r.categoryId);
    setIngredients(
      r.ingredients.length > 0
        ? r.ingredients.map((i) => ({ name: i.name, amount: i.amount, unit: i.unit }))
        : [{ name: "", amount: "", unit: "" }],
    );
    setSteps(
      r.steps.length > 0
        ? r.steps.map((s) => ({ instruction: s.instruction, durationSeconds: `${s.durationSeconds}` }))
        : [{ instruction: "", durationSeconds: "" }],
    );
  }

  const submit = () => {
    const payload = {
      title: title.trim(),
      description: description.trim(),
      categoryId,
      subcategoryId: "",
      servings: parseInt(servings) || 1,
      prepSeconds: (parseInt(prepMinutes) || 0) * 60,
      cookSeconds: (parseInt(cookMinutes) || 0) * 60,
      ingredients: ingredients
        .filter((i) => i.name.trim() !== "")
        .map((i) => ({ name: i.name.trim(), amount: i.amount.trim(), unit: i.unit.trim() })),
      steps: steps
        .filter((s) => s.instruction.trim() !== "")
        .map((s, idx) => ({
          position: idx + 1,
          instruction: s.instruction.trim(),
          durationSeconds: (parseInt(s.durationSeconds) || 0) * 60,
        })),
    };

    if (isNew) {
      createRecipe.mutate(payload, {
        onSuccess: (recipe) => {
          if (recipe) router.replace(`/(app)/recipe/${recipe.id}`);
        },
      });
    } else {
      updateRecipe.mutate(
        { ...payload, recipeId: id },
        {
          onSuccess: () => router.replace(`/(app)/recipe/${id}`),
        },
      );
    }
  };

  return (
    <SafeAreaView className="flex-1 bg-bg dark:bg-bg-dark" edges={["bottom"]}>
      <ScrollView contentContainerClassName="p-lg gap-md pb-2xl">
        <View className="flex-row items-center justify-between">
          <Text className="text-display font-bold text-fg dark:text-fg-dark">
            {isNew ? "New Recipe" : "Edit Recipe"}
          </Text>
          <Pressable accessibilityRole="button" onPress={() => router.back()}>
            <Text className="text-body text-muted dark:text-muted-dark">Cancel</Text>
          </Pressable>
        </View>

        <Field label="Title" value={title} onChangeText={setTitle} />

        <Field
          label="Description"
          value={description}
          onChangeText={setDescription}
          multiline
        />

        <View className="flex-row gap-md">
          <View className="flex-1">
            <Field label="Servings" value={servings} onChangeText={setServings} keyboardType="numeric" />
          </View>
          <View className="flex-1">
            <Field label="Prep (min)" value={prepMinutes} onChangeText={setPrepMinutes} keyboardType="numeric" />
          </View>
          <View className="flex-1">
            <Field label="Cook (min)" value={cookMinutes} onChangeText={setCookMinutes} keyboardType="numeric" />
          </View>
        </View>

        {/* Category picker */}
        {categories.data && categories.data.length > 0 && (
          <View className="gap-xs">
            <Text className="text-caption text-muted dark:text-muted-dark">Category</Text>
            <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerClassName="gap-xs">
              <CategoryChip label="None" active={categoryId === ""} onPress={() => setCategoryId("")} />
              {categories.data.map((cat) => (
                <CategoryChip
                  key={cat.id}
                  label={cat.name}
                  active={categoryId === cat.id}
                  onPress={() => setCategoryId(cat.id)}
                />
              ))}
            </ScrollView>
          </View>
        )}

        {/* Ingredients */}
        <View className="gap-xs">
          <Text className="text-title font-semibold text-fg dark:text-fg-dark">Ingredients</Text>
          {ingredients.map((ing, idx) => (
            <View key={idx} className="flex-row gap-sm">
              <View className="flex-1">
                <Field
                  label="Name"
                  value={ing.name}
                  onChangeText={(v) => updateRow(ingredients, setIngredients, idx, { name: v })}
                />
              </View>
              <View className="w-20">
                <Field
                  label="Amount"
                  value={ing.amount}
                  onChangeText={(v) => updateRow(ingredients, setIngredients, idx, { amount: v })}
                />
              </View>
              <View className="w-20">
                <Field
                  label="Unit"
                  value={ing.unit}
                  onChangeText={(v) => updateRow(ingredients, setIngredients, idx, { unit: v })}
                />
              </View>
            </View>
          ))}
          <Pressable
            accessibilityRole="button"
            onPress={() => setIngredients([...ingredients, { name: "", amount: "", unit: "" }])}
            className="items-center rounded-lg bg-card dark:bg-card-dark p-sm"
          >
            <Text className="text-caption text-primary">+ Add ingredient</Text>
          </Pressable>
        </View>

        {/* Steps */}
        <View className="gap-xs">
          <Text className="text-title font-semibold text-fg dark:text-fg-dark">Steps</Text>
          {steps.map((step, idx) => (
            <Card key={idx} className="gap-sm p-md">
              <View className="flex-row items-center gap-sm">
                <View className="h-7 w-7 items-center justify-center rounded-full bg-primary">
                  <Text className="text-caption text-primary-fg">{idx + 1}</Text>
                </View>
                <Text className="text-caption text-muted dark:text-muted-dark">Step {idx + 1}</Text>
              </View>
              <Field
                label="Instruction"
                value={step.instruction}
                onChangeText={(v) => updateRow(steps, setSteps, idx, { instruction: v })}
                multiline
              />
              <Field
                label="Duration (min)"
                value={step.durationSeconds}
                onChangeText={(v) => updateRow(steps, setSteps, idx, { durationSeconds: v })}
                keyboardType="numeric"
              />
            </Card>
          ))}
          <Pressable
            accessibilityRole="button"
            onPress={() => setSteps([...steps, { instruction: "", durationSeconds: "" }])}
            className="items-center rounded-lg bg-card dark:bg-card-dark p-sm"
          >
            <Text className="text-caption text-primary">+ Add step</Text>
          </Pressable>
        </View>

        <Button
          title={isNew ? "Create recipe" : "Save changes"}
          loading={createRecipe.isPending || updateRecipe.isPending}
          disabled={title.trim() === ""}
          onPress={() => submit()}
        />
      </ScrollView>
    </SafeAreaView>
  );
}

function updateRow<T extends object>(rows: T[], setRows: (r: T[]) => void, idx: number, patch: Partial<T>) {
  const next = [...rows];
  next[idx] = { ...next[idx], ...patch } as T;
  setRows(next);
}

function CategoryChip({
  label,
  active,
  onPress,
}: {
  label: string;
  active: boolean;
  onPress: () => void;
}) {
  return (
    <Pressable
      accessibilityRole="button"
      onPress={onPress}
      className={`rounded-full px-md py-sm ${active ? "bg-primary" : "bg-card dark:bg-card-dark"}`}
    >
      <Text className={`text-caption ${active ? "text-primary-fg" : "text-fg dark:text-fg-dark"}`}>
        {label}
      </Text>
    </Pressable>
  );
}