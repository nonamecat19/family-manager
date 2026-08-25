import Svg, { Circle, Path } from "react-native-svg";

import { organic } from "./tokens.ts";

/**
 * The icon set the design draws with — stroked 24×24 paths, no icon font. Kept here as
 * data so a screen never inlines an <Svg> of its own and drifts on stroke width.
 */
const PATHS = {
  home: "M4 11l8-6.5 8 6.5v8a1.5 1.5 0 0 1-1.5 1.5H14v-5h-4v5H5.5A1.5 1.5 0 0 1 4 19z",
  book: "M5 4.5A1.5 1.5 0 0 1 6.5 3H19v14.5H6.5A1.5 1.5 0 0 0 5 19zM19 17.5V21H6.5",
  plan: "M4 6.5A1.5 1.5 0 0 1 5.5 5h13A1.5 1.5 0 0 1 20 6.5v12A1.5 1.5 0 0 1 18.5 20h-13A1.5 1.5 0 0 1 4 18.5zM4 10h16M8.5 3v4M15.5 3v4",
  cart: "M4 5h2.2l2.3 10.2h8.6L19 8H7M9.5 20a1 1 0 1 0 0-.01M17 20a1 1 0 1 0 0-.01",
  user: "M12 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8zM5 20c1.2-3.3 3.8-5 7-5s5.8 1.7 7 5",
  back: "M15 5l-7 7 7 7",
  forward: "M9 5l7 7-7 7",
  close: "M6 6l12 12M18 6L6 18",
  filter: "M3 6h18M6 12h12M10 18h4",
  check: "M4 12.5l5 5L20 6.5",
  heart: "M12 20s-7-4.4-7-9a4 4 0 0 1 7-2.6A4 4 0 0 1 19 11c0 4.6-7 9-7 9z",
  plus: "M12 5v14M5 12h14",
  pencil: "M4 20h4l10-10-4-4L4 16zM14 6l4 4",
  trash: "M5 7h14M9 7V5h6v2M7 7l1 13h8l1-13",
} as const;

export type IconName = keyof typeof PATHS;

export function Icon({
  name,
  size = 22,
  color = organic.text,
  width = 2.75,
}: {
  name: IconName;
  size?: number;
  color?: string;
  width?: number;
}) {
  return (
    <Svg width={size} height={size} viewBox="0 0 24 24" fill="none">
      <Path
        d={PATHS[name]}
        stroke={color}
        strokeWidth={width}
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </Svg>
  );
}

/** Search is two shapes, so it can't ride on the single-path Icon. */
export function SearchIcon({ size = 19, color = organic.neutral[600] }: { size?: number; color?: string }) {
  return (
    <Svg width={size} height={size} viewBox="0 0 24 24" fill="none">
      <Circle cx={11} cy={11} r={7} stroke={color} strokeWidth={2.75} />
      <Path d="M20 20l-3.6-3.6" stroke={color} strokeWidth={2.75} strokeLinecap="round" />
    </Svg>
  );
}

export function ClockIcon({ size = 15, color = organic.neutral[700] }: { size?: number; color?: string }) {
  return (
    <Svg width={size} height={size} viewBox="0 0 24 24" fill="none">
      <Circle cx={12} cy={12} r={9} stroke={color} strokeWidth={2.75} />
      <Path d="M12 7v5l3 2" stroke={color} strokeWidth={2.75} strokeLinecap="round" />
    </Svg>
  );
}

/** A filled star — the rating mark. `filled={false}` draws the outline used by pickers. */
export function StarIcon({
  size = 13,
  filled = true,
  color = organic.accent.DEFAULT,
}: {
  size?: number;
  filled?: boolean;
  color?: string;
}) {
  return (
    <Svg width={size} height={size} viewBox="0 0 24 24">
      <Path
        d="M12 2l3 6.5 7 .9-5 4.8 1.2 7L12 17.8 5.8 21.2 7 14.2 2 9.4l7-.9z"
        fill={filled ? color : "none"}
        stroke={color}
        strokeWidth={filled ? 0 : 2}
        strokeLinejoin="round"
      />
    </Svg>
  );
}

/** The detail screen's favourite toggle: outlined when off, filled when on. */
export function HeartIcon({
  size = 19,
  filled = false,
  color = organic.accent[700],
}: {
  size?: number;
  filled?: boolean;
  color?: string;
}) {
  return (
    <Svg width={size} height={size} viewBox="0 0 24 24">
      <Path
        d={PATHS.heart}
        fill={filled ? organic.accent.DEFAULT : "none"}
        stroke={color}
        strokeWidth={2.5}
        strokeLinejoin="round"
      />
    </Svg>
  );
}

export function CheckIcon({ size = 13, color = "#ffffff" }: { size?: number; color?: string }) {
  return (
    <Svg width={size} height={size} viewBox="0 0 24 24" fill="none">
      <Path d={PATHS.check} stroke={color} strokeWidth={3.4} strokeLinecap="round" strokeLinejoin="round" />
    </Svg>
  );
}
