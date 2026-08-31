/**
 * Screen 10 — the widget gallery. Six widget types, each drawn at its home-screen footprint
 * with the household's live figures in it. Previews only: placing a widget is the Android
 * launcher's long-press flow, which is what the body copy tells the reader, so nothing on
 * this screen is tappable except the back button.
 */
import { toDisplayError } from "@fm/api";
import { useRouter } from "expo-router";
import { useCallback } from "react";
import { ScrollView, Text, View } from "react-native";

import { useI18n } from "@/components/i18n";
import { BottomIndicator, Button, Screen, ScreenHeader, monthName } from "@/components/nocturne";
import { useWidgetPreviews } from "@/components/screens/widgets/data.ts";
import { GalleryItem } from "@/components/screens/widgets/gallery.tsx";
import {
  AccountsPreview,
  BudgetsAndFamilyPreview,
  CategoryPreview,
  MonthPreview,
  QuickAddPreview,
  RecentPreview,
} from "@/components/screens/widgets/previews.tsx";

export default function WidgetsScreen() {
  const { t } = useI18n();
  const router = useRouter();
  const monthLabel = useCallback((month: number) => monthName(t, month), [t]);
  const { previews, isPending, error, refetch } = useWidgetPreviews(monthLabel);

  const size = (cols: number, rows: number) => t("widgets.size", { cols, rows });

  return (
    <Screen>
      <ScreenHeader
        gradient
        title={t("widgets.title")}
        leading={{ icon: "arrow-left", label: t("common.back"), onPress: () => router.back() }}
      >
        <Text className="mt-n3 text-[11px] leading-[17px] text-neutral-500">{t("widgets.body")}</Text>
      </ScreenHeader>

      {error ? (
        <Failure message={toDisplayError(error, t("common.loadFailed")).message} onRetry={refetch} />
      ) : !previews ? (
        <View className="flex-1 items-center justify-center">
          <Text className="text-[15px] text-neutral-500">
            {isPending ? t("common.loadingEllipsis") : t("common.none")}
          </Text>
        </View>
      ) : (
        <ScrollView
          className="flex-1 px-n4"
          contentContainerClassName="gap-n5 pb-n6 pt-n4"
          showsVerticalScrollIndicator={false}
        >
          <GalleryItem title={t("widgets.quickAdd")} size={size(4, 2)}>
            <QuickAddPreview
              model={previews.quickAdd}
              ownerLabel={t("widgets.templatesOf", {
                // A widget with no member behind it is the family-bound one the body copy
                // describes; naming it beats printing "Шаблони · ".
                name: previews.quickAdd.ownerName || t("common.family"),
              })}
              otherLabel={t("widgets.other")}
            />
          </GalleryItem>

          <View className="flex-row gap-n4">
            <GalleryItem title={t("widgets.month")} size={size(2, 2)} className="flex-1">
              <MonthPreview
                model={previews.month}
                overspentLabel={
                  previews.month.overspentCount > 0
                    ? t("common.budgetCount", { count: previews.month.overspentCount })
                    : null
                }
              />
            </GalleryItem>
            <GalleryItem title={t("widgets.category")} size={size(2, 1)} className="flex-1">
              <CategoryPreview model={previews.category} emptyLabel={t("common.none")} />
            </GalleryItem>
          </View>

          <GalleryItem title={t("widgets.budgetsAndFamily")} size={size(4, 3)}>
            <BudgetsAndFamilyPreview
              model={previews.budgetsAndFamily}
              emptyLabel={t("common.noBudget")}
            />
          </GalleryItem>

          <GalleryItem title={t("widgets.recent")} size={size(4, 2)}>
            <RecentPreview lines={previews.recent} emptyLabel={t("transactions.emptyTitle")} />
          </GalleryItem>

          <GalleryItem title={t("widgets.accounts")} size={size(4, 1)}>
            <AccountsPreview model={previews.accounts} emptyLabel={t("accounts.emptyTitle")} />
          </GalleryItem>
        </ScrollView>
      )}

      <BottomIndicator />
    </Screen>
  );
}

function Failure({ message, onRetry }: { message: string; onRetry: () => void }) {
  const { t } = useI18n();
  return (
    <View className="flex-1 justify-center gap-n4 px-n6">
      <Text className="text-[13.5px] leading-[21px] text-neutral-500">{message}</Text>
      <Button title={t("common.tryAgain")} onPress={onRetry} variant="ghost" />
    </View>
  );
}
