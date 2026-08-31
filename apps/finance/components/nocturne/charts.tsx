import { Text, View } from "react-native";
import Svg, { Circle, G } from "react-native-svg";

import { budgetState, nocturne, seriesColor } from "./tokens.ts";

/**
 * The four chart shapes the eleven screens use, in react-native-svg (already a repo
 * dependency) and plain views. No charting library: each of these is thirty lines of geometry,
 * and a library would be a stack row plus a theming layer to fight.
 *
 * Values are plain numbers in minor units. Charts do not format money — a caller labels them
 * with `MoneyText` or `formatMoney`, so the grammar stays in one place.
 */

// ---- donut -----------------------------------------------------------------------------------

export interface DonutSegment {
  id: string;
  label: string;
  /** Minor units, or any consistent unit. Negative and zero values are skipped. */
  value: number;
  /** Overrides the ramp slot; omit and the segment takes its colour from its index. */
  color?: string;
}

export interface DonutChartProps {
  segments: readonly DonutSegment[];
  /** Outer diameter. */
  size?: number;
  /** Ring width. */
  thickness?: number;
  /** The big figure in the middle — usually the period total, already formatted. */
  centerValue?: string;
  /** The line under it — "5 груп". */
  centerLabel?: string;
  /** Drilling into a group is a tap on its arc, as well as on its row. */
  onPressSegment?: (segment: DonutSegment) => void;
  /** Degrees of ground left between arcs. */
  gap?: number;
}

/** Home's group donut and Charts' own. Draws arcs as dashed circle strokes — one `Circle` per
 * segment, no path maths. */
export function DonutChart({
  segments,
  size = 196,
  thickness = 26,
  centerValue,
  centerLabel,
  onPressSegment,
  gap = 2,
}: DonutChartProps) {
  const drawable = segments.filter((s) => s.value > 0);
  const total = drawable.reduce((sum, s) => sum + s.value, 0);
  const radius = (size - thickness) / 2;
  const circumference = 2 * Math.PI * radius;

  let cursor = 0;
  return (
    <View style={{ width: size, height: size }} className="items-center justify-center">
      <Svg width={size} height={size}>
        {/* The unfilled ring, so an empty period still reads as a chart rather than a hole. */}
        <Circle
          cx={size / 2}
          cy={size / 2}
          r={radius}
          stroke={nocturne.neutral[900]}
          strokeWidth={thickness}
          fill="none"
        />
        {/* -90° puts the first segment at twelve o'clock, which is where the design starts. */}
        <G rotation={-90} origin={`${size / 2}, ${size / 2}`}>
          {total > 0
            ? drawable.map((segment, index) => {
                const fraction = segment.value / total;
                const length = Math.max(circumference * fraction - gap, 0.5);
                const offset = -cursor;
                cursor += circumference * fraction;
                return (
                  <Circle
                    key={segment.id}
                    cx={size / 2}
                    cy={size / 2}
                    r={radius}
                    stroke={segment.color ?? seriesColor(index)}
                    strokeWidth={thickness}
                    strokeLinecap="butt"
                    fill="none"
                    strokeDasharray={`${length} ${circumference - length}`}
                    strokeDashoffset={offset}
                    onPress={onPressSegment ? () => onPressSegment(segment) : undefined}
                  />
                );
              })
            : null}
        </G>
      </Svg>
      <View className="absolute items-center" pointerEvents="none">
        {centerValue ? <Text className="text-[24px] font-medium text-fg">{centerValue}</Text> : null}
        {centerLabel ? <Text className="mt-[2px] text-[11.5px] text-neutral-500">{centerLabel}</Text> : null}
      </View>
    </View>
  );
}

// ---- stacked bars ------------------------------------------------------------------------------

export interface BarSeries {
  id: string;
  label: string;
  color?: string;
}

export interface BarPoint {
  /** The axis label — "Сер". */
  label: string;
  /** One value per entry in `series`, same order. A short array is padded with zeros. */
  values: readonly number[];
}

export interface StackedBarSeriesProps {
  points: readonly BarPoint[];
  series: readonly BarSeries[];
  /** Plot height, excluding the axis labels and the legend. */
  height?: number;
  /** Dims every other column — the design highlights the month being viewed. */
  activeIndex?: number;
  onPressPoint?: (index: number) => void;
  legend?: boolean;
  className?: string;
}

/**
 * Charts' seven-month history, stacked by member. Built from views rather than SVG: the bars
 * are rectangles with rounded tops, and flexbox gets the column widths right at any screen
 * width without a measured viewport.
 */
export function StackedBarSeries({
  points,
  series,
  height = 150,
  activeIndex,
  onPressPoint,
  legend = true,
  className = "",
}: StackedBarSeriesProps) {
  const totals = points.map((p) => p.values.reduce((sum, v) => sum + Math.max(v, 0), 0));
  const peak = Math.max(...totals, 1);

  return (
    <View className={className}>
      <View className="flex-row items-end justify-between" style={{ height }}>
        {points.map((point, index) => {
          const dimmed = activeIndex != null && activeIndex !== index;
          return (
            <View
              key={`${point.label}-${index}`}
              className="flex-1 items-center"
              accessibilityRole={onPressPoint ? "button" : undefined}
              accessibilityLabel={point.label}
              onTouchEnd={onPressPoint ? () => onPressPoint(index) : undefined}
            >
              <View
                className="w-[62%] justify-end overflow-hidden rounded-sm"
                style={{ height: (totals[index] ?? 0) === 0 ? 2 : (height * (totals[index] ?? 0)) / peak }}
              >
                {series.map((s, si) => {
                  const value = Math.max(point.values[si] ?? 0, 0);
                  const share = (totals[index] ?? 0) === 0 ? 0 : value / (totals[index] ?? 1);
                  if (share === 0) return null;
                  return (
                    <View
                      key={s.id}
                      style={{
                        height: `${share * 100}%`,
                        backgroundColor: s.color ?? seriesColor(si),
                        opacity: dimmed ? 0.45 : 1,
                      }}
                    />
                  );
                })}
              </View>
            </View>
          );
        })}
      </View>
      <View className="mt-n3 flex-row justify-between">
        {points.map((point, index) => (
          <Text
            key={`${point.label}-label-${index}`}
            className={`flex-1 text-center text-[10.5px] ${
              activeIndex === index ? "text-fg" : "text-neutral-600"
            }`}
          >
            {point.label}
          </Text>
        ))}
      </View>
      {legend ? <ChartLegend series={series} className="mt-n4" /> : null}
    </View>
  );
}

export function ChartLegend({
  series,
  className = "",
}: {
  series: readonly BarSeries[];
  className?: string;
}) {
  return (
    <View className={`flex-row flex-wrap gap-n4 ${className}`}>
      {series.map((s, index) => (
        <View key={s.id} className="flex-row items-center gap-n2">
          <View
            className="h-[8px] w-[8px] rounded-full"
            style={{ backgroundColor: s.color ?? seriesColor(index) }}
          />
          <Text className="text-[11.5px] text-neutral-400">{s.label}</Text>
        </View>
      ))}
    </View>
  );
}

// ---- split bar ------------------------------------------------------------------------------------

export interface SplitPart {
  id: string;
  label?: string;
  value: number;
  color?: string;
}

export interface SplitBarProps {
  parts: readonly SplitPart[];
  height?: number;
  /** Rounds the whole bar rather than each part — the design's "Розподіл" is one capsule. */
  rounded?: boolean;
  className?: string;
}

/** The per-member split: one bar cut into shares. Two parts on screen 06, but it takes any
 * number. */
export function SplitBar({ parts, height = 10, rounded = true, className = "" }: SplitBarProps) {
  const total = parts.reduce((sum, p) => sum + Math.max(p.value, 0), 0);
  return (
    <View
      className={`w-full flex-row overflow-hidden ${className}`}
      style={{ height, borderRadius: rounded ? height / 2 : 0, backgroundColor: nocturne.neutral[900] }}
    >
      {total > 0
        ? parts.map((part, index) => (
            <View
              key={part.id}
              accessibilityLabel={part.label}
              style={{
                flexGrow: Math.max(part.value, 0),
                flexBasis: 0,
                backgroundColor: part.color ?? seriesColor(index),
              }}
            />
          ))
        : null}
    </View>
  );
}

// ---- budget bar --------------------------------------------------------------------------------------

export interface BudgetBarProps {
  /** Minor units. */
  spentMinor: number;
  /** Minor units. `0` means "no budget": no bar is drawn at all, which is not the same as
   * "spent nothing". */
  limitMinor: number;
  /** The row's name, drawn above the bar. Omit to draw the bar alone. */
  label?: string;
  /** Right-hand text. Defaults to the computed percentage — "98%", "256%". */
  valueLabel?: string;
  height?: number;
  /** Overrides the fill colour while under budget; the overspend state always wins. */
  color?: string;
  className?: string;
}

/**
 * A group's budget. Under 100% the fill is the accent and grows with the ratio; past 100% the
 * bar fills completely in `overspend` and the figure turns with it — the one thing in this app
 * that has to read at a glance.
 */
export function BudgetBar({
  spentMinor,
  limitMinor,
  label,
  valueLabel,
  height = 6,
  color,
  className = "",
}: BudgetBarProps) {
  const state = budgetState(spentMinor, limitMinor);
  const over = state?.over ?? false;
  const fraction = state ? Math.min(state.ratio, 1) : 0;
  const shown = valueLabel ?? (state ? `${state.percent}%` : "");

  return (
    <View className={className}>
      {label || shown ? (
        <View className="mb-n2 flex-row items-baseline justify-between">
          {label ? <Text className="text-[12.5px] text-neutral-300">{label}</Text> : <View />}
          {shown ? (
            <Text className={`text-[12px] font-medium ${over ? "text-overspend" : "text-neutral-400"}`}>
              {shown}
            </Text>
          ) : null}
        </View>
      ) : null}
      <View
        className="w-full overflow-hidden"
        style={{ height, borderRadius: height / 2, backgroundColor: nocturne.neutral[900] }}
      >
        <View
          style={{
            width: `${fraction * 100}%`,
            height: "100%",
            borderRadius: height / 2,
            backgroundColor: over ? nocturne.overspend : (color ?? nocturne.accent.DEFAULT),
          }}
        />
      </View>
    </View>
  );
}
