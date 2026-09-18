import { toDisplayError } from "@fm/api";
import { useRouter } from "expo-router";
import { useCallback } from "react";
import { Text, View } from "react-native";

import { useI18n } from "@/components/i18n";
import { monthName } from "@/components/kit";
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
import { Button, Screen, ScreenHeader, ScrollBody } from "@fm/ui";

export default function WidgetsScreen() {
  const { t } = useI18n();
  const router = useRouter();
  const monthLabel = useCallback((month: number) => monthName(t, month), [t]);
  const { previews, isPending, error, refetch } = useWidgetPreviews(monthLabel);

  const size = (cols: number, rows: number) => t("widgets.size", { cols, rows });

  return (
    <Screen>
      <ScrollBody>
        <ScreenHeader
          title={t("widgets.title")}
          onBack={() => router.back()}
          backLabel={t("common.back")}
        />
        <Text className="-mt-sm text-[14px] leading-[21px] text-neutral-600">{t("widgets.body")}</Text>

        {error ? (
          <Failure message={toDisplayError(error, t("common.loadFailed")).message} onRetry={refetch} />
        ) : !previews ? (
          <View className="items-center py-7">
            <Text className="text-[15px] text-neutral-600">
              {isPending ? t("common.loadingEllipsis") : t("common.none")}
            </Text>
          </View>
        ) : (
          <>
            <GalleryItem title={t("widgets.quickAdd")} size={size(4, 2)}>
              <QuickAddPreview
                model={previews.quickAdd}
                ownerLabel={t("widgets.templatesOf", {
                  name: previews.quickAdd.ownerName || t("common.family"),
                })}
                otherLabel={t("widgets.other")}
              />
            </GalleryItem>

            <View className="flex-row gap-3">
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
          </>
        )}
      </ScrollBody>
    </Screen>
  );
}

function Failure({ message, onRetry }: { message: string; onRetry: () => void }) {
  const { t } = useI18n();
  return (
    <View className="gap-3 py-4.5">
      <Text className="text-[13.5px] leading-[21px] text-neutral-600">{message}</Text>
      <Button title={t("common.tryAgain")} onPress={onRetry} tone="quiet" />
    </View>
  );
}
