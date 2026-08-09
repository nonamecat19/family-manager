import {
  useRecipe,
  useComments,
  useAddComment,
  useToggleFavorite,
  useDeleteRecipe,
} from "@fm/api";
import { Button, Card, ErrorState, Field, Loading } from "@fm/ui";
import { useLocalSearchParams, useRouter } from "expo-router";
import { useState } from "react";
import { Alert, FlatList, Image, KeyboardAvoidingView, Platform, Pressable, ScrollView, Text, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

export default function RecipeDetailScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const router = useRouter();
  const recipe = useRecipe(id);
  const comments = useComments(id);
  const addComment = useAddComment();
  const toggleFavorite = useToggleFavorite();
  const deleteRecipe = useDeleteRecipe();

  const [commentBody, setCommentBody] = useState("");

  if (recipe.isPending) return <Loading label="Loading recipe…" />;
  if (recipe.isError) return <ErrorState message={recipe.error.message} onRetry={() => void recipe.refetch()} />;
  const r = recipe.data;
  if (!r) return null;

  const totalSeconds = r.prepSeconds + r.cookSeconds;

  const handleDelete = () => {
    Alert.alert("Delete recipe?", "This cannot be undone.", [
      { text: "Cancel", style: "cancel" },
      {
        text: "Delete",
        style: "destructive",
        onPress: () => {
          deleteRecipe.mutate(r.id, {
            onSuccess: () => router.replace("/(app)"),
          });
        },
      },
    ]);
  };

  return (
    <SafeAreaView className="flex-1 bg-bg dark:bg-bg-dark" edges={["bottom"]}>
      <KeyboardAvoidingView
        behavior={Platform.OS === "ios" ? "padding" : undefined}
        className="flex-1"
      >
        <ScrollView contentContainerClassName="p-lg gap-md pb-2xl">
          {r.imageUrl !== "" && (
            <Image
              source={{ uri: r.imageUrl }}
              className="h-56 w-full rounded-lg bg-card dark:bg-card-dark"
              resizeMode="cover"
            />
          )}

          {/* Header */}
          <View className="gap-xs">
            <Text className="text-display font-bold text-fg dark:text-fg-dark">{r.title}</Text>
            {r.description !== "" && (
              <Text className="text-body text-muted dark:text-muted-dark">{r.description}</Text>
            )}
          </View>

          {/* Stats */}
          <Card className="flex-row justify-around p-md">
            <Stat label="Servings" value={`${r.servings}`} />
            {r.prepSeconds > 0 && <Stat label="Prep" value={formatDuration(r.prepSeconds)} />}
            {r.cookSeconds > 0 && <Stat label="Cook" value={formatDuration(r.cookSeconds)} />}
            {totalSeconds > 0 && <Stat label="Total" value={formatDuration(totalSeconds)} />}
          </Card>

          {/* Actions */}
          <View className="flex-row gap-md">
            <Pressable
              accessibilityRole="button"
              onPress={() => toggleFavorite.mutate(r.id)}
              className="flex-1 items-center rounded-lg bg-card dark:bg-card-dark p-md"
            >
              <Text className="text-body text-fg dark:text-fg-dark">
                {r.favoriteCount > 0 ? "♥ Favorited" : "♥ Favorite"}
              </Text>
            </Pressable>
            <Pressable
              accessibilityRole="button"
              onPress={() => router.push(`/(app)/recipe-edit/${r.id}`)}
              className="flex-1 items-center rounded-lg bg-card dark:bg-card-dark p-md"
            >
              <Text className="text-body text-fg dark:text-fg-dark">✎ Edit</Text>
            </Pressable>
          </View>

          {/* Ingredients */}
          {r.ingredients.length > 0 && (
            <View className="gap-xs">
              <Text className="text-title font-semibold text-fg dark:text-fg-dark">Ingredients</Text>
              {r.ingredients.map((ing, i) => (
                <View key={i} className="flex-row justify-between rounded-lg bg-card dark:bg-card-dark px-md py-sm">
                  <Text className="text-body text-fg dark:text-fg-dark">{ing.name}</Text>
                  <Text className="text-body text-muted dark:text-muted-dark">
                    {ing.amount} {ing.unit}
                  </Text>
                </View>
              ))}
            </View>
          )}

          {/* Steps */}
          {r.steps.length > 0 && (
            <View className="gap-xs">
              <Text className="text-title font-semibold text-fg dark:text-fg-dark">Steps</Text>
              {r.steps.map((step, i) => (
                <Card key={i} className="gap-xs p-md">
                  <View className="flex-row items-center gap-sm">
                    <View className="h-7 w-7 items-center justify-center rounded-full bg-primary">
                      <Text className="text-caption text-primary-fg">{step.position}</Text>
                    </View>
                    <Text className="flex-1 text-body text-fg dark:text-fg-dark">
                      {step.instruction}
                    </Text>
                  </View>
                  {step.durationSeconds > 0 && (
                    <Text className="ml-9 text-caption text-muted dark:text-muted-dark">
                      ⏱ {formatDuration(step.durationSeconds)}
                    </Text>
                  )}
                </Card>
              ))}
            </View>
          )}

          {/* Comments */}
          <View className="gap-xs">
            <Text className="text-title font-semibold text-fg dark:text-fg-dark">
              Comments {r.commentCount > 0 && `(${r.commentCount})`}
            </Text>

            {comments.data && comments.data.length > 0 ? (
              <FlatList
                data={comments.data}
                keyExtractor={(c) => c.id}
                scrollEnabled={false}
                renderItem={({ item: c }) => (
                  <Card className="gap-xs p-md">
                    <Text className="text-caption text-muted dark:text-muted-dark">
                      {new Date(Number(c.createdAt!.seconds) * 1000).toLocaleDateString()}
                    </Text>
                    <Text className="text-body text-fg dark:text-fg-dark">{c.body}</Text>
                  </Card>
                )}
              />
            ) : (
              <Text className="text-caption text-muted dark:text-muted-dark">
                No comments yet.
              </Text>
            )}

            {/* Add comment */}
            <View className="flex-row gap-sm">
              <View className="flex-1">
                <Field
                  label="Add a comment"
                  value={commentBody}
                  onChangeText={setCommentBody}
                  multiline
                />
              </View>
              <Button
                title="Post"
                disabled={commentBody.trim() === "" || addComment.isPending}
                onPress={() => {
                  addComment.mutate(
                    { recipeId: r.id, body: commentBody.trim() },
                    { onSuccess: () => setCommentBody("") },
                  );
                }}
              />
            </View>
          </View>

          {/* Delete */}
          <Pressable
            accessibilityRole="button"
            onPress={handleDelete}
            className="items-center rounded-lg bg-card dark:bg-card-dark p-md"
          >
            <Text className="text-body text-error">Delete recipe</Text>
          </Pressable>
        </ScrollView>
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <View className="items-center gap-xs">
      <Text className="text-caption text-muted dark:text-muted-dark">{label}</Text>
      <Text className="text-body font-semibold text-fg dark:text-fg-dark">{value}</Text>
    </View>
  );
}

function formatDuration(seconds: number): string {
  if (seconds < 60) return `${seconds}s`;
  const mins = Math.floor(seconds / 60);
  if (mins < 60) return `${mins}m`;
  const hrs = Math.floor(mins / 60);
  return `${hrs}h ${mins % 60}m`;
}