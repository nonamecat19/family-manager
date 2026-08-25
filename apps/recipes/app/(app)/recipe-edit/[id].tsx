import {
  useCreateRecipe,
  useRecipe,
  useRecipeCategories,
  useRecipeSubcategories,
  useUpdateRecipe,
  useUploadRecipeImage,
} from "@fm/api";
import { useLocalSearchParams, useRouter } from "expo-router";
import * as ImagePicker from "expo-image-picker";
import { useState } from "react";
import { Image, Pressable, ScrollView, Text, View } from "react-native";

import { Icon } from "../../../components/organic/icons.tsx";
import { organic } from "../../../components/organic/tokens.ts";
import {
  Chip,
  DashedButton,
  Display,
  Field,
  Kicker,
  PrimaryButton,
  RoundButton,
  Screen,
  StarPicker,
} from "../../../components/organic/ui.tsx";

interface IngredientRow {
  name: string;
  amount: string;
  unit: string;
}

interface StepRow {
  instruction: string;
  durationMinutes: string;
}

export default function RecipeEditScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const router = useRouter();
  const isNew = id === "new";
  const recipe = useRecipe(isNew ? "" : id);
  const categories = useRecipeCategories();
  const createRecipe = useCreateRecipe();
  const updateRecipe = useUpdateRecipe();
  const uploadImage = useUploadRecipeImage();

  const [pickedImage, setPickedImage] = useState<{ uri: string; base64: string; contentType: string } | null>(
    null,
  );
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [servings, setServings] = useState("1");
  const [prepMinutes, setPrepMinutes] = useState("0");
  const [cookMinutes, setCookMinutes] = useState("0");
  const [categoryId, setCategoryId] = useState("");
  const [subcategoryId, setSubcategoryId] = useState("");
  const [notes, setNotes] = useState("");
  const [rating, setRating] = useState(0);
  const [ingredients, setIngredients] = useState<IngredientRow[]>([{ name: "", amount: "", unit: "" }]);
  const [steps, setSteps] = useState<StepRow[]>([{ instruction: "", durationMinutes: "" }]);

  const subcategories = useRecipeSubcategories(categoryId);

  if (!isNew && recipe.isPending) return <Placeholder label="Opening the recipe…" />;
  if (!isNew && recipe.isError) {
    return <Placeholder label={recipe.error.message} />;
  }
  // Load the existing recipe into the form once, on the render where it first arrives.
  if (!isNew && recipe.data && title === "" && recipe.data.title !== "") {
    const r = recipe.data;
    setTitle(r.title);
    setDescription(r.description);
    setServings(`${r.servings}`);
    setPrepMinutes(`${Math.floor(r.prepSeconds / 60)}`);
    setCookMinutes(`${Math.floor(r.cookSeconds / 60)}`);
    setCategoryId(r.categoryId);
    setSubcategoryId(r.subcategoryId);
    setNotes(r.notes);
    setRating(r.rating);
    setIngredients(
      r.ingredients.length > 0
        ? r.ingredients.map((i) => ({ name: i.name, amount: i.amount, unit: i.unit }))
        : [{ name: "", amount: "", unit: "" }],
    );
    setSteps(
      r.steps.length > 0
        ? r.steps.map((s) => ({
            instruction: s.instruction,
            durationMinutes: s.durationSeconds > 0 ? `${Math.round(s.durationSeconds / 60)}` : "",
          }))
        : [{ instruction: "", durationMinutes: "" }],
    );
  }

  const pickImage = async () => {
    const perm = await ImagePicker.requestMediaLibraryPermissionsAsync();
    if (!perm.granted) return;
    // quality < 1 or allowsEditing both make expo-image-picker re-encode the asset as JPEG on
    // Android/iOS, which has no alpha channel — a background-removed PNG would come back with
    // a black or white background baked in. Passing the original file through untouched is the
    // only way to keep transparency, since the backend already stores whatever bytes it gets.
    const result = await ImagePicker.launchImageLibraryAsync({
      mediaTypes: ImagePicker.MediaTypeOptions.Images,
      base64: true,
      quality: 1,
    });
    const asset = result.assets?.[0];
    if (result.canceled || !asset?.base64) return;
    setPickedImage({ uri: asset.uri, base64: asset.base64, contentType: asset.mimeType ?? "image/jpeg" });
  };

  // The recipe must exist before an image can be attached to it, so a picked photo is
  // uploaded as a second request once create/update has returned an id.
  const uploadPickedImage = (recipeId: string) => {
    if (!pickedImage) return;
    uploadImage.mutate({
      recipeId,
      imageData: base64ToBytes(pickedImage.base64),
      contentType: pickedImage.contentType,
    });
  };

  const submit = () => {
    const payload = {
      title: title.trim(),
      description: description.trim(),
      categoryId,
      subcategoryId,
      servings: parseInt(servings) || 1,
      prepSeconds: (parseInt(prepMinutes) || 0) * 60,
      cookSeconds: (parseInt(cookMinutes) || 0) * 60,
      notes: notes.trim(),
      rating,
      ingredients: ingredients
        .filter((i) => i.name.trim() !== "")
        .map((i) => ({ name: i.name.trim(), amount: i.amount.trim(), unit: i.unit.trim() })),
      steps: steps
        .filter((s) => s.instruction.trim() !== "")
        .map((s, idx) => ({
          position: idx + 1,
          instruction: s.instruction.trim(),
          durationSeconds: (parseInt(s.durationMinutes) || 0) * 60,
        })),
    };

    if (isNew) {
      createRecipe.mutate(payload, {
        onSuccess: (created) => {
          if (!created) return;
          uploadPickedImage(created.id);
          router.replace(`/(app)/recipe/${created.id}`);
        },
      });
    } else {
      updateRecipe.mutate(
        { ...payload, recipeId: id },
        {
          onSuccess: () => {
            uploadPickedImage(id);
            router.replace(`/(app)/recipe/${id}`);
          },
        },
      );
    }
  };

  const existingImage = !isNew ? recipe.data?.imageUrl : undefined;

  return (
    <Screen edges={["bottom"]}>
      <ScrollView
        keyboardShouldPersistTaps="handled"
        showsVerticalScrollIndicator={false}
        contentContainerClassName="gap-[20px] px-[22px] pb-[32px] pt-[8px]"
      >
        <View className="flex-row items-center gap-[13px]">
          <RoundButton icon="close" label="Cancel" onPress={() => router.back()} />
          <Display size={24}>{isNew ? "New recipe" : "Edit recipe"}</Display>
        </View>

        <View className="flex-row items-center gap-[16px]">
          <Pressable
            accessibilityRole="button"
            accessibilityLabel="Add a photo"
            onPress={() => void pickImage()}
            className="h-[104px] w-[104px] flex-none items-center justify-center overflow-hidden rounded-full bg-accent-200"
          >
            {pickedImage || (existingImage !== undefined && existingImage !== "") ? (
              <Image
                source={{ uri: pickedImage?.uri ?? existingImage }}
                className="h-[104px] w-[104px]"
                resizeMode="contain"
              />
            ) : (
              <Icon name="plus" size={26} color={organic.accent[700]} />
            )}
          </Pressable>
          <Text className="flex-1 font-fig-semi text-[13.5px] leading-[20px] text-neutral-600">
            A transparent PNG works best — the dish sits straight on the page with no box
            around it.
          </Text>
        </View>

        <Field label="Name" value={title} onChangeText={setTitle} placeholder="Babcia's mushroom soup" />
        <Field
          label="Description"
          value={description}
          onChangeText={setDescription}
          placeholder="What is it, in one line"
        />

        {categories.data && categories.data.length > 0 && (
          <View>
            <Kicker className="mb-[10px]">Category</Kicker>
            <View className="flex-row flex-wrap gap-[8px]">
              <Chip
                label="None"
                active={categoryId === ""}
                onPress={() => {
                  setCategoryId("");
                  setSubcategoryId("");
                }}
              />
              {categories.data.map((c) => (
                <Chip
                  key={c.id}
                  label={c.name}
                  active={categoryId === c.id}
                  onPress={() => {
                    setCategoryId(c.id);
                    setSubcategoryId("");
                  }}
                />
              ))}
            </View>
          </View>
        )}

        {categoryId !== "" && subcategories.data && subcategories.data.length > 0 && (
          <View>
            <Kicker className="mb-[10px]">Subcategory</Kicker>
            <View className="flex-row flex-wrap gap-[8px]">
              <Chip
                label="None"
                active={subcategoryId === ""}
                onPress={() => setSubcategoryId("")}
                tone="accent2"
              />
              {subcategories.data.map((s) => (
                <Chip
                  key={s.id}
                  label={s.name}
                  active={subcategoryId === s.id}
                  onPress={() => setSubcategoryId(s.id)}
                  tone="accent2"
                />
              ))}
            </View>
          </View>
        )}

        <View className="flex-row gap-[12px]">
          <Field
            className="flex-1"
            label="Servings"
            value={servings}
            onChangeText={setServings}
            keyboardType="numeric"
          />
          <Field
            className="flex-1"
            label="Prep (min)"
            value={prepMinutes}
            onChangeText={setPrepMinutes}
            keyboardType="numeric"
          />
          <Field
            className="flex-1"
            label="Cook (min)"
            value={cookMinutes}
            onChangeText={setCookMinutes}
            keyboardType="numeric"
          />
        </View>

        <View>
          <Kicker className="mb-[10px]">Rating</Kicker>
          <StarPicker rating={rating} onChange={setRating} />
        </View>

        <View>
          <Kicker className="mb-[10px]">Ingredients</Kicker>
          <View className="gap-[8px]">
            {ingredients.map((ing, idx) => (
              <View key={idx} className="flex-row items-end gap-[8px]">
                <Field
                  className="flex-1"
                  label={idx === 0 ? "Name" : ""}
                  value={ing.name}
                  onChangeText={(v) => updateRow(ingredients, setIngredients, idx, { name: v })}
                  placeholder="Chestnut mushrooms"
                />
                <Field
                  className="w-[76px]"
                  label={idx === 0 ? "Amount" : ""}
                  value={ing.amount}
                  onChangeText={(v) => updateRow(ingredients, setIngredients, idx, { amount: v })}
                  placeholder="300"
                />
                <Field
                  className="w-[64px]"
                  label={idx === 0 ? "Unit" : ""}
                  value={ing.unit}
                  onChangeText={(v) => updateRow(ingredients, setIngredients, idx, { unit: v })}
                  placeholder="g"
                />
              </View>
            ))}
            <DashedButton
              title="+ Add ingredient"
              onPress={() => setIngredients([...ingredients, { name: "", amount: "", unit: "" }])}
            />
          </View>
        </View>

        <View>
          <Kicker className="mb-[10px]">Steps</Kicker>
          <View className="gap-[10px]">
            {steps.map((step, idx) => (
              <View key={idx} className="gap-[8px] rounded-2xl bg-neutral-100 px-[14px] py-[14px]">
                <View className="flex-row items-center gap-[10px]">
                  <View className="h-[26px] w-[26px] items-center justify-center rounded-full bg-accent2-300">
                    <Text className="font-cap text-[13px] text-accent2-900">{idx + 1}</Text>
                  </View>
                  <Text className="font-fig-bold text-[12.5px] text-neutral-600">Step {idx + 1}</Text>
                </View>
                <Field
                  label="Instruction"
                  value={step.instruction}
                  onChangeText={(v) => updateRow(steps, setSteps, idx, { instruction: v })}
                  multiline
                  placeholder="Sweat the onion and carrot in butter until soft."
                />
                <Field
                  label="Timer (min)"
                  value={step.durationMinutes}
                  onChangeText={(v) => updateRow(steps, setSteps, idx, { durationMinutes: v })}
                  keyboardType="numeric"
                  placeholder="0"
                />
              </View>
            ))}
            <DashedButton
              title="+ Add step"
              onPress={() => setSteps([...steps, { instruction: "", durationMinutes: "" }])}
            />
          </View>
        </View>

        <Field
          label="Notes"
          value={notes}
          onChangeText={setNotes}
          multiline
          placeholder="Babcia never measured the cream."
        />

        <PrimaryButton
          title={
            createRecipe.isPending || updateRecipe.isPending
              ? "Saving…"
              : isNew
                ? "Save recipe"
                : "Save changes"
          }
          disabled={title.trim() === "" || createRecipe.isPending || updateRecipe.isPending}
          onPress={submit}
        />
      </ScrollView>
    </Screen>
  );
}

function Placeholder({ label }: { label: string }) {
  return (
    <Screen>
      <View className="flex-1 items-center justify-center px-[22px]">
        <Text className="text-center font-fig-semi text-[14px] text-neutral-600">{label}</Text>
      </View>
    </Screen>
  );
}

function updateRow<T extends object>(rows: T[], setRows: (r: T[]) => void, idx: number, patch: Partial<T>) {
  const next = [...rows];
  next[idx] = { ...next[idx], ...patch } as T;
  setRows(next);
}

const BASE64_ALPHABET = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";

// RN's JS engine has no built-in atob, so ImagePicker's base64 string is decoded by hand
// rather than pulling in a polyfill for one call site.
function base64ToBytes(base64: string): Uint8Array {
  const clean = base64.replace(/=+$/, "");
  const bytes = new Uint8Array(Math.floor((clean.length * 3) / 4));
  let byteIndex = 0;
  for (let i = 0; i < clean.length; i += 4) {
    const c0 = BASE64_ALPHABET.indexOf(clean[i] ?? "=");
    const c1 = BASE64_ALPHABET.indexOf(clean[i + 1] ?? "=");
    const c2 = BASE64_ALPHABET.indexOf(clean[i + 2] ?? "=");
    const c3 = BASE64_ALPHABET.indexOf(clean[i + 3] ?? "=");
    const chunk = (c0 << 18) | (c1 << 12) | ((c2 & 63) << 6) | (c3 & 63);
    bytes[byteIndex++] = (chunk >> 16) & 0xff;
    if (clean[i + 2] !== undefined) bytes[byteIndex++] = (chunk >> 8) & 0xff;
    if (clean[i + 3] !== undefined) bytes[byteIndex++] = chunk & 0xff;
  }
  return bytes;
}
