import Svg, { Path } from "react-native-svg";

import { nocturne } from "./tokens.ts";

/**
 * The icon set, drawn as 24×24 stroked paths after Phosphor — the family the imported design
 * uses (it loads `@phosphor-icons/web`, which is a WEB ICON FONT and cannot render in React
 * Native).
 *
 * Why not a package: apps/recipes and apps/finance already answer this question. Each carries
 * its own hand-drawn path set rather than an icon dependency. Adding `phosphor-react-native`
 * would put a row in the stack that docs/stack.md does not carry, for ~40 glyphs, and a
 * dependency addition is a gate this task may not walk through.
 *
 * Kept here as data so a screen never inlines an <Svg> of its own and drifts on stroke width.
 */
const PATHS = {
  // ---- chrome & navigation ----------------------------------------------------------------
  "magnifying-glass": "M10.5 17a6.5 6.5 0 1 0 0-13 6.5 6.5 0 0 0 0 13zM15.5 15.5L20.5 20.5",
  plus: "M12 5v14M5 12h14",
  "plus-circle": "M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18zM8 12h8M12 8v8",
  "arrow-left": "M20 12H4M10 6l-6 6 6 6",
  "caret-left": "M14.5 6l-6 6 6 6",
  "caret-down": "M6 9.5l6 6 6-6",
  "caret-up": "M6 14.5l6-6 6 6",
  "caret-up-down": "M8 10l4-4 4 4M8 14l4 4 4-4",
  "dots-three": "M6 12h.01M12 12h.01M18 12h.01",
  "dots-six-vertical": "M9.5 7h.01M9.5 12h.01M9.5 17h.01M14.5 7h.01M14.5 12h.01M14.5 17h.01",
  x: "M6 6l12 12M18 6L6 18",
  "gear-six":
    "M12 15.5a3.5 3.5 0 1 0 0-7 3.5 3.5 0 0 0 0 7zM19.4 13.5l1.8 1.2-1.9 3.3-2.1-.7a7.6 7.6 0 0 1-1.8 1L15 20.5h-3.8l-.4-2.2a7.6 7.6 0 0 1-1.8-1l-2.1.7-1.9-3.3 1.8-1.2a7.7 7.7 0 0 1 0-3L3 8.3l1.9-3.3 2.1.7a7.6 7.6 0 0 1 1.8-1l.4-2.2H15l.4 2.2a7.6 7.6 0 0 1 1.8 1l2.1-.7L21.2 8.3l-1.8 1.2a7.7 7.7 0 0 1 0 4z",

  // ---- the rail ----------------------------------------------------------------------------
  notebook: "M6 3h13v18H6a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1zM9.5 3v18M12.5 8h4M12.5 12h4",
  "folder-simple":
    "M3 6.5A1.5 1.5 0 0 1 4.5 5h3.9l2.4 3h8.7A1.5 1.5 0 0 1 21 9.5v9a1.5 1.5 0 0 1-1.5 1.5h-15A1.5 1.5 0 0 1 3 18.5z",
  archive: "M3.5 4.5h17V9h-17zM5 9v10.5A1.5 1.5 0 0 0 6.5 21h11a1.5 1.5 0 0 0 1.5-1.5V9M9.5 13h5",
  star: "M12 3.2l2.7 5.6 6.1.8-4.4 4.3 1.1 6.1L12 17.1 6.5 20l1.1-6.1L3.2 9.6l6.1-.8z",
  "clock-counter-clockwise": "M12 7.5v5l4 2M3.6 9.4A9 9 0 1 1 3 13.5M3.6 4.5v4.9h4.9",

  // ---- people & sharing ---------------------------------------------------------------------
  "users-three":
    "M12 10.5a3 3 0 1 0 0-6 3 3 0 0 0 0 6zM7.5 19c.9-2.3 2.5-3.5 4.5-3.5s3.6 1.2 4.5 3.5M5.5 13.5a2.5 2.5 0 1 1 0-5M2 18c.5-1.4 1.4-2.3 2.7-2.7M18.5 8.5a2.5 2.5 0 1 1 0 5M22 18c-.5-1.4-1.4-2.3-2.7-2.7",
  users:
    "M9 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8zM2.5 19c1.3-3.2 3.6-4.8 6.5-4.8s5.2 1.6 6.5 4.8M16 4.5a4 4 0 0 1 0 7.5M17 14.4c2.2.5 3.7 2 4.5 4.1",
  "user-plus": "M10 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8zM3 20c1.3-3.2 3.8-4.8 7-4.8M17 13.5v5.5M14.2 16.2h5.6",
  "chat-teardrop-text": "M12 20.5a8.5 8.5 0 1 0-8.5-8.5v7.5c0 .6.4 1 1 1zM8.5 10.5h7M8.5 14h4.5",

  // ---- list controls -------------------------------------------------------------------------
  "sort-ascending": "M4 7h10M4 12h7M4 17h4M17.5 5v14M17.5 19l3-3M17.5 19l-3-3",
  "funnel-simple": "M4 6h16M7 12h10M10 18h4",
  // The density toggle's other face: the roomy card row, against list-bullets' one-liners.
  rows: "M4 5.5h16v5.5H4zM4 13h16v5.5H4z",

  // ---- sync state -----------------------------------------------------------------------------
  "cloud-check": "M7 19a4.5 4.5 0 0 1-.4-9 6 6 0 0 1 11.4 1.2A4 4 0 0 1 17 19zM9.6 13.6l2 2 3.6-3.8",
  "cloud-slash": "M7 19a4.5 4.5 0 0 1-.4-9 6 6 0 0 1 11.4 1.2A4 4 0 0 1 17 19zM4 4l16 16",

  // ---- blocks & the editor's block bar ----------------------------------------------------------
  check: "M4.5 12.5l5 5L19.5 7",
  "check-square":
    "M4.5 6A1.5 1.5 0 0 1 6 4.5h12A1.5 1.5 0 0 1 19.5 6v12a1.5 1.5 0 0 1-1.5 1.5H6A1.5 1.5 0 0 1 4.5 18zM8.5 12l2.5 2.5 4.5-5",
  "file-text": "M6 3.5h7.5L18 8v12.5H6zM13.5 3.5V8H18M9 12h6M9 15.5h6M9 8.5h2.5",
  "list-bullets": "M9 6h11M9 12h11M9 18h11M4.5 6h.01M4.5 12h.01M4.5 18h.01",
  "list-numbers": "M9.5 6.5h10.5M9.5 12h10.5M9.5 17.5h10.5M3.5 5.5l1.5-1v5M3 18.5h2.8M3 15.6a1.4 1.4 0 1 1 2.4 1L3 19",
  minus: "M4.5 12h15",
  quotes:
    "M9.5 7c-2.5 1.2-4 3.4-4 6.2V17H11v-5.5H8c0-1.4.5-2.4 1.5-3zM18.5 7c-2.5 1.2-4 3.4-4 6.2V17H20v-5.5h-3c0-1.4.5-2.4 1.5-3z",
  code: "M9 7l-5 5 5 5M15 7l5 5-5 5",
  image:
    "M4 6.5A1.5 1.5 0 0 1 5.5 5h13A1.5 1.5 0 0 1 20 6.5v11a1.5 1.5 0 0 1-1.5 1.5h-13A1.5 1.5 0 0 1 4 17.5zM4 15.5l4.5-4 4 3.5 3-2.5 4.5 4M15 9.5h.01",
  paperclip:
    "M17.5 10.5l-6.4 6.4a3.5 3.5 0 0 1-5-5l7.5-7.5a2.5 2.5 0 0 1 3.5 3.5l-7.4 7.4a1.5 1.5 0 0 1-2.1-2.1l6.4-6.4",
  microphone: "M12 15a3.5 3.5 0 0 0 3.5-3.5v-5a3.5 3.5 0 0 0-7 0v5A3.5 3.5 0 0 0 12 15zM5.5 11.5a6.5 6.5 0 0 0 13 0M12 18v3M9 21h6",
  camera: "M4 8.5h3.5L9 6h6l1.5 2.5H20v10H4zM12 16a3.2 3.2 0 1 0 0-6.4 3.2 3.2 0 0 0 0 6.4z",
  "link-simple": "M10 8H7.5a4 4 0 0 0 0 8H10M14 8h2.5a4 4 0 0 1 0 8H14M8.5 12h7",
  trash: "M4 6.5h16M9.5 6.5V4.5h5v2M6.5 6.5l1 13.5h9l1-13.5M10 10.5v6M14 10.5v6",
  "pencil-simple": "M4.5 19.5h4l11-11a2.1 2.1 0 0 0-3-3l-11 11zM14.5 6.5l3 3",

  // ---- the formatting bar ------------------------------------------------------------------------
  "text-aa": "M3 18l4.5-11L12 18M4.7 14.5h5.6M21 11v7M21 12.6a3.4 3.4 0 1 0 0 4.6",
  "text-b": "M7 5h5.5a3.4 3.4 0 0 1 0 6.8H7zM7 11.8h6.3a3.6 3.6 0 0 1 0 7.2H7z",
  "text-italic": "M10 5h8M6 19h8M14.5 5l-5 14",
  "text-h-two": "M4 6v12M4 12h7M11 6v12M15.5 9a2.8 2.8 0 0 1 5 1.7c0 2.5-5 3.6-5 7.3h5",

  // ---- misc ----------------------------------------------------------------------------------------
  circle: "M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18z",
} as const;

/**
 * The glyphs that have a real silhouette, for `weight="fill"`. Only the shapes the design
 * actually fills are here — a filled star is a different mark from an outlined one, and the
 * list draws both. Anything else asked for filled falls back to a heavier stroke, which at
 * the 14–22px these are drawn reads the same.
 */
const FILLED_PATHS: Partial<Record<IconName, string>> = {
  star: PATHS.star,
  circle: PATHS.circle,
  "check-square": PATHS["check-square"],
};

export type IconName = keyof typeof PATHS;
export type IconWeight = "regular" | "fill";

export const ICON_NAMES = Object.keys(PATHS) as IconName[];

export interface IconProps {
  name: IconName;
  /** Square side in px. The design draws rail glyphs at 18, block-bar glyphs at 20–22. */
  size?: number;
  color?: string;
  weight?: IconWeight;
}

export function Icon({ name, size = 20, color = nocturne.text, weight = "regular" }: IconProps) {
  const filled = weight === "fill" ? FILLED_PATHS[name] : undefined;
  return (
    <Svg width={size} height={size} viewBox="0 0 24 24" fill="none">
      <Path
        d={filled ?? PATHS[name]}
        fill={filled ? color : "none"}
        stroke={color}
        strokeWidth={filled ? 0 : weight === "fill" ? 2.3 : 1.7}
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </Svg>
  );
}

/** Narrows an arbitrary string (a stored icon key) to a drawable name. */
export function isIconName(value: string | null | undefined): value is IconName {
  return value != null && Object.hasOwn(PATHS, value);
}

/** The icon a row falls back to when its stored key is unknown or empty. */
export const FALLBACK_ICON: IconName = "file-text";

export function iconOr(value: string | null | undefined, fallback: IconName = FALLBACK_ICON): IconName {
  return isIconName(value) ? value : fallback;
}
