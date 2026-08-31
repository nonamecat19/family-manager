import Svg, { Path } from "react-native-svg";

import { nocturne } from "./tokens.ts";

/**
 * The icon set, drawn as 24×24 stroked paths after Phosphor — the family the design canvas
 * uses (it loads `@phosphor-icons/web`, which is a WEB ICON FONT and cannot render in React
 * Native).
 *
 * Why not a package: apps/recipes already answers this question. It carries its own
 * `components/organic/icons.tsx` of hand-drawn paths rather than an icon dependency, and the
 * brief is explicit that an app should reuse that approach where it exists. Adding
 * `phosphor-react-native` would put a row in the stack that docs/stack.md does not carry, for
 * ~60 glyphs, and dependency additions are a gate this task may not walk through.
 *
 * `weight="fill"` — the design's category glyphs, which sit inside a solid tinted circle — is
 * approximated by a heavier stroke rather than a second set of silhouette paths. At the 17–24px
 * the design draws them, inside a filled circle, the two read the same; a real silhouette set
 * would double this file for no legibility gain.
 */
const PATHS = {
  // ---- navigation & chrome ----------------------------------------------------------------
  list: "M4 7h16M4 12h16M4 17h16",
  "caret-down": "M6 9.5l6 6 6-6",
  "caret-up": "M6 14.5l6-6 6 6",
  "caret-left": "M14.5 6l-6 6 6 6",
  "caret-right": "M9.5 6l6 6-6 6",
  "arrow-left": "M20 12H4M10 6l-6 6 6 6",
  "arrow-right": "M4 12h16M14 6l6 6-6 6",
  plus: "M12 5v14M5 12h14",
  x: "M6 6l12 12M18 6L6 18",
  check: "M4.5 12.5l5 5L19.5 7",
  "dots-three": "M6 12h.01M12 12h.01M18 12h.01",
  "magnifying-glass": "M10.5 17a6.5 6.5 0 1 0 0-13 6.5 6.5 0 0 0 0 13zM15.5 15.5L20.5 20.5",
  funnel: "M4 5h16l-6 7v6.5l-4 2V12z",
  gear: "M12 15.5a3.5 3.5 0 1 0 0-7 3.5 3.5 0 0 0 0 7zM19.4 13.5l1.8 1.2-1.9 3.3-2.1-.7a7.6 7.6 0 0 1-1.8 1L15 20.5h-3.8l-.4-2.2a7.6 7.6 0 0 1-1.8-1l-2.1.7-1.9-3.3 1.8-1.2a7.7 7.7 0 0 1 0-3L3 8.3l1.9-3.3 2.1.7a7.6 7.6 0 0 1 1.8-1l.4-2.2H15l.4 2.2a7.6 7.6 0 0 1 1.8 1l2.1-.7L21.2 8.3l-1.8 1.2a7.7 7.7 0 0 1 0 4z",
  "arrows-out-line-vertical": "M8 8l4-4 4 4M8 16l4 4 4-4M4 12h16",
  "sliders-horizontal": "M4 7h8M16 7h4M4 12h4M12 12h8M4 17h8M16 17h4M14 5a2 2 0 1 1 0 4 2 2 0 0 1 0-4M10 10a2 2 0 1 1 0 4 2 2 0 0 1 0-4M14 15a2 2 0 1 1 0 4 2 2 0 0 1 0-4",
  "squares-four": "M4 4.5h6.5V11H4zM13.5 4.5H20V11h-6.5zM4 13.5h6.5V20H4zM13.5 13.5H20V20h-6.5z",
  "pencil-simple": "M4 20h4L20 8l-4-4L4 16zM14.5 5.5l4 4",
  tray: "M4 15h4l1.5 2.5h5L16 15h4M5 15l2.5-9h9L19 15v4a1.5 1.5 0 0 1-1.5 1.5h-11A1.5 1.5 0 0 1 5 19z",

  // ---- status bar --------------------------------------------------------------------------
  "bell-slash": "M6 10a6 6 0 0 1 9.6-4.8M18 12.5V10a6 6 0 0 0-.4-2.2M6 10c0 4-1.5 5.5-1.5 5.5h12M10 19a2 2 0 0 0 4 0M4 4l16 16",
  "wifi-high": "M4.5 10a11 11 0 0 1 15 0M7.5 13.2a6.5 6.5 0 0 1 9 0M12 17.5h.01",
  "cell-signal-medium": "M4 20h3.5v-5H4zM10 20h3.5V9H10zM16.5 20H20V4h-3.5z",

  // ---- time & date -------------------------------------------------------------------------
  clock: "M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18zM12 7v5l3.5 2",
  "clock-counter-clockwise": "M12 7.5v5l4 2M3.6 9.4A9 9 0 1 1 3 13.5M3.6 4.5v4.9h4.9",
  "calendar-dots":
    "M4 7.5A1.5 1.5 0 0 1 5.5 6h13A1.5 1.5 0 0 1 20 7.5v11A1.5 1.5 0 0 1 18.5 20h-13A1.5 1.5 0 0 1 4 18.5zM4 10h16M8 4v4M16 4v4M8.5 14h.01M12 14h.01M15.5 14h.01",
  calculator:
    "M5.5 4.5A1.5 1.5 0 0 1 7 3h10a1.5 1.5 0 0 1 1.5 1.5v15A1.5 1.5 0 0 1 17 21H7a1.5 1.5 0 0 1-1.5-1.5zM8.5 7.5h7M8.5 12h.01M12 12h.01M15.5 12h.01M8.5 16h.01M12 16h.01M15.5 16h.01",

  // ---- people ------------------------------------------------------------------------------
  "users-three":
    "M12 10.5a3 3 0 1 0 0-6 3 3 0 0 0 0 6zM7.5 19c.9-2.3 2.5-3.5 4.5-3.5s3.6 1.2 4.5 3.5M5.5 13.5a2.5 2.5 0 1 1 0-5M2 18c.5-1.4 1.4-2.3 2.7-2.7M18.5 8.5a2.5 2.5 0 1 1 0 5M22 18c-.5-1.4-1.4-2.3-2.7-2.7",
  "user-plus": "M10 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8zM3 20c1.3-3.2 3.8-4.8 7-4.8M17 13.5v5.5M14.2 16.2h5.6",
  user: "M12 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8zM5 20c1.2-3.3 3.8-5 7-5s5.8 1.7 7 5",

  // ---- money & accounts ---------------------------------------------------------------------
  wallet:
    "M4 7.5A2.5 2.5 0 0 1 6.5 5H17v3.5M4 7.5v9A2.5 2.5 0 0 0 6.5 19H19v-3.5M20 10.5h-4.5a2 2 0 0 0 0 4H20z",
  "credit-card": "M3 7.5A1.5 1.5 0 0 1 4.5 6h15A1.5 1.5 0 0 1 21 7.5v9a1.5 1.5 0 0 1-1.5 1.5h-15A1.5 1.5 0 0 1 3 16.5zM3 10h18M6.5 14.5h3",
  money: "M2.5 7h19v10h-19zM12 14.5a2.5 2.5 0 1 0 0-5 2.5 2.5 0 0 0 0 5zM6 10.5h.01M18 13.5h.01",
  "piggy-bank":
    "M3.5 12.5c0-3.6 3.6-6.5 8-6.5 1.3 0 2.5.2 3.6.6L18.5 5v3.4c1 .9 1.6 2 1.8 3.1H22v3.5h-2c-.5 1.4-1.5 2.6-2.8 3.4V20h-3.2v-1.3a11 11 0 0 1-3.5 0V20H7.3v-1.9C5 16.9 3.5 14.9 3.5 12.5zM15.5 12h.01M9.5 6.4C9 4.8 9.6 3.5 9.6 3.5s2 .4 2.7 1.7",
  "currency-btc": "M7 5v14M7 5h6.2a3 3 0 0 1 0 6H7M7 11h7a3.5 3.5 0 0 1 0 7H7M10.5 2.5v3M14 2.5v3M10.5 18v3.5M14 18v3.5",
  receipt: "M5 3.5l2 1.6 2-1.6 2 1.6 2-1.6 2 1.6 2-1.6v17l-2-1.6-2 1.6-2-1.6-2 1.6-2-1.6-2 1.6zM8.5 9.5h7M8.5 14h7",
  "arrows-left-right": "M3 9h18M7 5L3 9l4 4M21 15H3M17 11l4 4-4 4",
  target: "M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18zM12 16.5a4.5 4.5 0 1 0 0-9 4.5 4.5 0 0 0 0 9zM12 13.5a1.5 1.5 0 1 0 0-3 1.5 1.5 0 0 0 0 3z",

  // ---- charts -------------------------------------------------------------------------------
  "chart-donut": "M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18zM12 16a4 4 0 1 0 0-8 4 4 0 0 0 0 8zM12 3v5M18.4 16.5l-4.3-2.5",
  "chart-bar": "M4 20h16M7 20v-8M12 20V6M17 20v-5",
  "trend-up": "M4 17l6-6 3.5 3.5L21 7M15 7h6v6",

  // ---- privacy & system ----------------------------------------------------------------------
  "eye-slash":
    "M4 4l16 16M9.6 9.7a3 3 0 0 0 4.2 4.2M6.4 6.6C3.9 8.2 2.5 10.6 2 12c1.3 3.3 5 7 10 7 1.8 0 3.4-.5 4.8-1.2M9.8 5.3A9.6 9.6 0 0 1 12 5c5 0 8.7 3.7 10 7-.6 1.5-1.7 3.2-3.3 4.6",
  "lock-key": "M6 10.5h12V20H6zM8.5 10.5V7a3.5 3.5 0 0 1 7 0v3.5M12 14a1.5 1.5 0 1 0 0 3 1.5 1.5 0 0 0 0-3zM12 17v1.5",
  "lock-simple": "M5.5 10h13v10h-13zM8.5 10V7a3.5 3.5 0 0 1 7 0v3",
  palette:
    "M12 20a8 8 0 1 1 0-16c4.4 0 8 3.1 8 7 0 2.2-1.8 3.5-4 3.5h-1.6c-1.2 0-2.2 1-2.2 2.2 0 .4.1.8.3 1.1.2.3.3.7.3 1.1 0 .6-.4 1.1-.8 1.1zM8 9.5h.01M12 7h.01M16 9.5h.01M7.5 14h.01",
  database:
    "M12 8.5c4.4 0 8-1.2 8-2.75S16.4 3 12 3 4 4.2 4 5.75 7.6 8.5 12 8.5zM4 5.75v12.5C4 19.8 7.6 21 12 21s8-1.2 8-2.75V5.75M4 12c0 1.55 3.6 2.75 8 2.75s8-1.2 8-2.75",
  bell: "M6 10a6 6 0 1 1 12 0c0 4 1.5 5.5 1.5 5.5h-15S6 14 6 10zM10 19a2 2 0 0 0 4 0",
  "bell-ringing": "M6 10a6 6 0 1 1 12 0c0 4 1.5 5.5 1.5 5.5h-15S6 14 6 10zM10 19a2 2 0 0 0 4 0M2.8 8.2a5.5 5.5 0 0 1 2.4-4M21.2 8.2a5.5 5.5 0 0 0-2.4-4",
  lightning: "M13.5 3L5 13.5h6L10.5 21 19 10.5h-6z",
  "arrows-clockwise": "M20 8.5V4M20 8.5h-4.5M4 15.5V20M4 15.5h4.5M19.6 8.5A8 8 0 0 0 5 9.6M4.4 15.5a8 8 0 0 0 14.4-1.1",
  "arrow-clockwise": "M19.5 9.5V4.8M19.5 9.5h-4.7M19.2 9.5A8 8 0 1 0 20 13",
  "folder-plus":
    "M4 6.5A1.5 1.5 0 0 1 5.5 5h3.6l2 2.5h7.4A1.5 1.5 0 0 1 20 9v9.5a1.5 1.5 0 0 1-1.5 1.5h-13A1.5 1.5 0 0 1 4 18.5zM12 11.5v6M9 14.5h6",

  // ---- category glyphs (drawn filled inside a tinted circle) ----------------------------------
  "graduation-cap": "M2.5 9.5L12 5l9.5 4.5L12 14zM6.5 11.4V16c0 1.7 2.5 3 5.5 3s5.5-1.3 5.5-3v-4.6M20.5 10.2V16",
  "house-line": "M4 20.5h16M5.5 20.5v-9.7L12 5.5l6.5 5.3v9.7M10 20.5v-5h4v5",
  basket: "M3 9.5h18l-1.7 9.1a2 2 0 0 1-2 1.4H6.7a2 2 0 0 1-2-1.4zM7.5 9.5L11 4M16.5 9.5L13 4M9.5 13v3.5M14.5 13v3.5",
  scroll: "M4 6.5A2.5 2.5 0 0 1 6.5 4H19v13a3 3 0 0 0 3 3H7a3 3 0 0 1-3-3zM4 6.5A2.5 2.5 0 0 0 6.5 9H9M12 8h4M12 12h4",
  coffee: "M4 8h12v6.5a4.5 4.5 0 0 1-9 0V8zM16 9.5h1.8a2.6 2.6 0 0 1 0 5.2H16M5 20h13M8.5 5.5V3.5M12 5.5V3.5",
  "fork-knife": "M7 3v7a2.75 2.75 0 0 0 5.5 0V3M9.75 12.5V21M17.5 3c-1.6 1.4-2.5 3.3-2.5 5.5 0 2 1 3.3 2.5 3.5V21",
  wine: "M7 4h10l-.6 5.5A4.5 4.5 0 0 1 12 14a4.5 4.5 0 0 1-4.4-4.5zM12 14v5M8.5 19h7M7.3 8h9.4",
  cookie: "M20.9 11.6A9 9 0 1 1 12.4 3.1a3.6 3.6 0 0 0 4.8 4.8 3.6 3.6 0 0 0 3.7 3.7zM9 9h.01M8 14.5h.01M13 15h.01M14 11h.01",
  drop: "M12 21a7 7 0 0 0 7-7c0-5-7-11-7-11S5 9 5 14a7 7 0 0 0 7 7z",
  pizza: "M12 3c4.3 0 8.1 2.5 9.9 6.2L12 21 2.1 9.2C3.9 5.5 7.7 3 12 3zM4.4 8.6A17 17 0 0 1 12 6.9c2.8 0 5.4.6 7.6 1.7M10 11.5h.01M13.5 14.5h.01",
  "bowl-food": "M3 11h18v.5a9 9 0 0 1-18 0zM7.5 8.5a3 3 0 0 1 3-2.5M13 6a3 3 0 0 1 3 2.5M6 19h12",
  bus: "M5 5.5A1.5 1.5 0 0 1 6.5 4h11A1.5 1.5 0 0 1 19 5.5v11a1.5 1.5 0 0 1-1.5 1.5h-11A1.5 1.5 0 0 1 5 16.5zM5 11h14M8 14.5h.01M16 14.5h.01M8 18v2M16 18v2M3.5 8v4M20.5 8v4",
  car: "M3.5 16.5v-4.2l2-5A2 2 0 0 1 7.4 6h9.2a2 2 0 0 1 1.9 1.3l2 5v4.2zM3.5 16.5h17M5.5 16.5V19H8v-2.5M16 16.5V19h2.5v-2.5M5.5 11.5h13M7.5 14H9M15 14h1.5",
  "car-profile": "M3 16v-3.5l2-4.5A2 2 0 0 1 6.8 7H15l3.5 5 2.5.8V16zM3 16h18M7.5 14a2 2 0 1 1 0 4 2 2 0 0 1 0-4M17 14a2 2 0 1 1 0 4 2 2 0 0 1 0-4M5.2 12.5h11",
  heartbeat: "M20.5 9.5A4.7 4.7 0 0 0 12 6.8 4.7 4.7 0 0 0 3.6 9.5M3 12.5h3.5l2-4 3 8 2.5-6 1.5 2H21M4.5 14.5c1.7 2.9 5.6 5 7.5 6.5 1.9-1.5 5.8-3.6 7.5-6.5",
  "game-controller":
    "M8 6.5h8a5.5 5.5 0 0 1 5.4 6.5l-.6 3.4A2.8 2.8 0 0 1 15.4 18L14 16h-4l-1.4 2a2.8 2.8 0 0 1-4.4-1.6L3.6 13A5.5 5.5 0 0 1 8 6.5zM8.5 11h-3M7 9.5v3M15.5 10h.01M17.5 12h.01",
  "device-mobile": "M6.5 3.5h11v17h-11zM10 5.8h4",
} as const;

export type IconName = keyof typeof PATHS;
export type IconWeight = "regular" | "fill";

export const ICON_NAMES = Object.keys(PATHS) as IconName[];

export interface IconProps {
  name: IconName;
  /** Square side in px. The design draws chrome at 18–22 and category glyphs at 17–24. */
  size?: number;
  color?: string;
  weight?: IconWeight;
}

export function Icon({ name, size = 20, color = nocturne.text, weight = "regular" }: IconProps) {
  return (
    <Svg width={size} height={size} viewBox="0 0 24 24" fill="none">
      <Path
        d={PATHS[name]}
        stroke={color}
        strokeWidth={weight === "fill" ? 2.3 : 1.7}
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </Svg>
  );
}

/** Narrows an arbitrary string (a category's stored icon key) to a drawable name. */
export function isIconName(value: string | null | undefined): value is IconName {
  return value != null && Object.hasOwn(PATHS, value);
}

/** The icon a category/group falls back to when its stored key is unknown or empty. */
export const FALLBACK_ICON: IconName = "dots-three";

export function iconOr(value: string | null | undefined, fallback: IconName = FALLBACK_ICON): IconName {
  return isIconName(value) ? value : fallback;
}
