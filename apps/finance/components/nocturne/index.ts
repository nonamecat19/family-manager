/**
 * The Nocturne kit's single door. Screens import from "@/components/nocturne" and nothing
 * else under this folder — the file layout inside is free to move.
 */

export {
  nocturne,
  SERIES,
  FONT_NOTE,
  seriesColor,
  memberColor,
  tintFor,
  initialOf,
  budgetState,
  type Tint,
  type BudgetState,
} from "./tokens.ts";

export {
  Icon,
  ICON_NAMES,
  FALLBACK_ICON,
  isIconName,
  iconOr,
  type IconName,
  type IconProps,
  type IconWeight,
} from "./icons.tsx";

export {
  currencySymbol,
  formatMinor,
  formatMoney,
  formatCompact,
  formatPercent,
  zeroLike,
  MINUS,
  type MoneyFormatOptions,
} from "./money.ts";

export {
  parseISO,
  isoWeekday,
  monthName,
  monthNameLower,
  monthNameGenitive,
  monthNameShort,
  monthTitle,
  dayHeading,
  relativeDay,
  shortDate,
  type Translate,
} from "./format.ts";

export {
  Screen,
  ScreenHeader,
  IconButton,
  Kicker,
  Divider,
  BottomIndicator,
  Card,
  ListSection,
  Row,
  MoneyText,
  MemberAvatar,
  IconCircle,
  Badge,
  Chip,
  Button,
  Fab,
  Field,
  ToggleRow,
  Stat,
  EmptyState,
  Sheet,
  type HeaderAction,
  type ScreenHeaderProps,
  type CardProps,
  type ListSectionProps,
  type RowProps,
  type MoneyTextProps,
  type MoneyTone,
  type MemberAvatarProps,
  type IconCircleProps,
  type BadgeTone,
  type ChipProps,
  type ButtonProps,
  type FieldProps,
  type EmptyStateProps,
  type SheetProps,
} from "./ui.tsx";

export {
  SegmentedTabs,
  PeriodTabs,
  PERIOD_TABS,
  PeriodStepper,
  MonthStepper,
  ScopeSwitcher,
  scopeIsFamily,
  type SegmentedOption,
  type SegmentedTabsProps,
  type PeriodTab,
  type PeriodTabsProps,
  type PeriodStepperProps,
  type MonthStepperProps,
  type Scope,
  type ScopeOption,
  type ScopeSwitcherProps,
} from "./tabs.tsx";

export {
  DonutChart,
  StackedBarSeries,
  ChartLegend,
  SplitBar,
  BudgetBar,
  type DonutSegment,
  type DonutChartProps,
  type BarSeries,
  type BarPoint,
  type StackedBarSeriesProps,
  type SplitPart,
  type SplitBarProps,
  type BudgetBarProps,
} from "./charts.tsx";

export { CategoryIconGrid, type CategoryGridItem, type CategoryIconGridProps } from "./grid.tsx";

export {
  Drawer,
  useDrawerItems,
  type DrawerProps,
  type DrawerItem,
  type DrawerScope,
} from "./drawer.tsx";
