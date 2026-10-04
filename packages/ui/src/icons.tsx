import Svg, { Circle, Path } from "react-native-svg";

import { organic } from "./tokens.ts";

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
  "magnifying-glass": "M10.5 17a6.5 6.5 0 1 0 0-13 6.5 6.5 0 0 0 0 13zM15.5 15.5L20.5 20.5",
  "plus-circle": "M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18zM8 12h8M12 8v8",
  "arrow-left": "M20 12H4M10 6l-6 6 6 6",
  "caret-left": "M14.5 6l-6 6 6 6",
  "caret-down": "M6 9.5l6 6 6-6",
  "caret-up": "M6 14.5l6-6 6 6",
  "caret-up-down": "M8 10l4-4 4 4M8 14l4 4 4-4",
  "dots-three": "M6 12h.01M12 12h.01M18 12h.01",
  "dots-six-vertical": "M9.5 7h.01M9.5 12h.01M9.5 17h.01M14.5 7h.01M14.5 12h.01M14.5 17h.01",
  x: "M6 6l12 12M18 6L6 18",
  "gear-six": "M12 15.5a3.5 3.5 0 1 0 0-7 3.5 3.5 0 0 0 0 7zM19.4 13.5l1.8 1.2-1.9 3.3-2.1-.7a7.6 7.6 0 0 1-1.8 1L15 20.5h-3.8l-.4-2.2a7.6 7.6 0 0 1-1.8-1l-2.1.7-1.9-3.3 1.8-1.2a7.7 7.7 0 0 1 0-3L3 8.3l1.9-3.3 2.1.7a7.6 7.6 0 0 1 1.8-1l.4-2.2H15l.4 2.2a7.6 7.6 0 0 1 1.8 1l2.1-.7L21.2 8.3l-1.8 1.2a7.7 7.7 0 0 1 0 4z",
  notebook: "M6 3h13v18H6a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1zM9.5 3v18M12.5 8h4M12.5 12h4",
  "folder-simple": "M3 6.5A1.5 1.5 0 0 1 4.5 5h3.9l2.4 3h8.7A1.5 1.5 0 0 1 21 9.5v9a1.5 1.5 0 0 1-1.5 1.5h-15A1.5 1.5 0 0 1 3 18.5z",
  archive: "M3.5 4.5h17V9h-17zM5 9v10.5A1.5 1.5 0 0 0 6.5 21h11a1.5 1.5 0 0 0 1.5-1.5V9M9.5 13h5",
  star: "M12 3.2l2.7 5.6 6.1.8-4.4 4.3 1.1 6.1L12 17.1 6.5 20l1.1-6.1L3.2 9.6l6.1-.8z",
  "clock-counter-clockwise": "M12 7.5v5l4 2M3.6 9.4A9 9 0 1 1 3 13.5M3.6 4.5v4.9h4.9",
  "users-three": "M12 10.5a3 3 0 1 0 0-6 3 3 0 0 0 0 6zM7.5 19c.9-2.3 2.5-3.5 4.5-3.5s3.6 1.2 4.5 3.5M5.5 13.5a2.5 2.5 0 1 1 0-5M2 18c.5-1.4 1.4-2.3 2.7-2.7M18.5 8.5a2.5 2.5 0 1 1 0 5M22 18c-.5-1.4-1.4-2.3-2.7-2.7",
  users: "M9 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8zM2.5 19c1.3-3.2 3.6-4.8 6.5-4.8s5.2 1.6 6.5 4.8M16 4.5a4 4 0 0 1 0 7.5M17 14.4c2.2.5 3.7 2 4.5 4.1",
  "user-plus": "M10 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8zM3 20c1.3-3.2 3.8-4.8 7-4.8M17 13.5v5.5M14.2 16.2h5.6",
  "chat-teardrop-text": "M12 20.5a8.5 8.5 0 1 0-8.5-8.5v7.5c0 .6.4 1 1 1zM8.5 10.5h7M8.5 14h4.5",
  "sort-ascending": "M4 7h10M4 12h7M4 17h4M17.5 5v14M17.5 19l3-3M17.5 19l-3-3",
  "funnel-simple": "M4 6h16M7 12h10M10 18h4",
  rows: "M4 5.5h16v5.5H4zM4 13h16v5.5H4z",
  "cloud-check": "M7 19a4.5 4.5 0 0 1-.4-9 6 6 0 0 1 11.4 1.2A4 4 0 0 1 17 19zM9.6 13.6l2 2 3.6-3.8",
  "cloud-slash": "M7 19a4.5 4.5 0 0 1-.4-9 6 6 0 0 1 11.4 1.2A4 4 0 0 1 17 19zM4 4l16 16",
  "check-square": "M4.5 6A1.5 1.5 0 0 1 6 4.5h12A1.5 1.5 0 0 1 19.5 6v12a1.5 1.5 0 0 1-1.5 1.5H6A1.5 1.5 0 0 1 4.5 18zM8.5 12l2.5 2.5 4.5-5",
  "file-text": "M6 3.5h7.5L18 8v12.5H6zM13.5 3.5V8H18M9 12h6M9 15.5h6M9 8.5h2.5",
  "list-bullets": "M9 6h11M9 12h11M9 18h11M4.5 6h.01M4.5 12h.01M4.5 18h.01",
  "list-numbers": "M9.5 6.5h10.5M9.5 12h10.5M9.5 17.5h10.5M3.5 5.5l1.5-1v5M3 18.5h2.8M3 15.6a1.4 1.4 0 1 1 2.4 1L3 19",
  minus: "M4.5 12h15",
  quotes: "M9.5 7c-2.5 1.2-4 3.4-4 6.2V17H11v-5.5H8c0-1.4.5-2.4 1.5-3zM18.5 7c-2.5 1.2-4 3.4-4 6.2V17H20v-5.5h-3c0-1.4.5-2.4 1.5-3z",
  code: "M9 7l-5 5 5 5M15 7l5 5-5 5",
  image: "M4 6.5A1.5 1.5 0 0 1 5.5 5h13A1.5 1.5 0 0 1 20 6.5v11a1.5 1.5 0 0 1-1.5 1.5h-13A1.5 1.5 0 0 1 4 17.5zM4 15.5l4.5-4 4 3.5 3-2.5 4.5 4M15 9.5h.01",
  paperclip: "M17.5 10.5l-6.4 6.4a3.5 3.5 0 0 1-5-5l7.5-7.5a2.5 2.5 0 0 1 3.5 3.5l-7.4 7.4a1.5 1.5 0 0 1-2.1-2.1l6.4-6.4",
  microphone: "M12 15a3.5 3.5 0 0 0 3.5-3.5v-5a3.5 3.5 0 0 0-7 0v5A3.5 3.5 0 0 0 12 15zM5.5 11.5a6.5 6.5 0 0 0 13 0M12 18v3M9 21h6",
  camera: "M4 8.5h3.5L9 6h6l1.5 2.5H20v10H4zM12 16a3.2 3.2 0 1 0 0-6.4 3.2 3.2 0 0 0 0 6.4z",
  "link-simple": "M10 8H7.5a4 4 0 0 0 0 8H10M14 8h2.5a4 4 0 0 1 0 8H14M8.5 12h7",
  "pencil-simple": "M4.5 19.5h4l11-11a2.1 2.1 0 0 0-3-3l-11 11zM14.5 6.5l3 3",
  "text-aa": "M3 18l4.5-11L12 18M4.7 14.5h5.6M21 11v7M21 12.6a3.4 3.4 0 1 0 0 4.6",
  "text-b": "M7 5h5.5a3.4 3.4 0 0 1 0 6.8H7zM7 11.8h6.3a3.6 3.6 0 0 1 0 7.2H7z",
  "text-italic": "M10 5h8M6 19h8M14.5 5l-5 14",
  "text-h-two": "M4 6v12M4 12h7M11 6v12M15.5 9a2.8 2.8 0 0 1 5 1.7c0 2.5-5 3.6-5 7.3h5",
  circle: "M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18z",
  list: "M4 7h16M4 12h16M4 17h16",
  "caret-right": "M9.5 6l6 6-6 6",
  "arrow-right": "M4 12h16M14 6l6 6-6 6",
  funnel: "M4 5h16l-6 7v6.5l-4 2V12z",
  gear: "M12 15.5a3.5 3.5 0 1 0 0-7 3.5 3.5 0 0 0 0 7zM19.4 13.5l1.8 1.2-1.9 3.3-2.1-.7a7.6 7.6 0 0 1-1.8 1L15 20.5h-3.8l-.4-2.2a7.6 7.6 0 0 1-1.8-1l-2.1.7-1.9-3.3 1.8-1.2a7.7 7.7 0 0 1 0-3L3 8.3l1.9-3.3 2.1.7a7.6 7.6 0 0 1 1.8-1l.4-2.2H15l.4 2.2a7.6 7.6 0 0 1 1.8 1l2.1-.7L21.2 8.3l-1.8 1.2a7.7 7.7 0 0 1 0 4z",
  "arrows-out-line-vertical": "M8 8l4-4 4 4M8 16l4 4 4-4M4 12h16",
  "sliders-horizontal": "M4 7h8M16 7h4M4 12h4M12 12h8M4 17h8M16 17h4M14 5a2 2 0 1 1 0 4 2 2 0 0 1 0-4M10 10a2 2 0 1 1 0 4 2 2 0 0 1 0-4M14 15a2 2 0 1 1 0 4 2 2 0 0 1 0-4",
  "squares-four": "M4 4.5h6.5V11H4zM13.5 4.5H20V11h-6.5zM4 13.5h6.5V20H4zM13.5 13.5H20V20h-6.5z",
  tray: "M4 15h4l1.5 2.5h5L16 15h4M5 15l2.5-9h9L19 15v4a1.5 1.5 0 0 1-1.5 1.5h-11A1.5 1.5 0 0 1 5 19z",
  "bell-slash": "M6 10a6 6 0 0 1 9.6-4.8M18 12.5V10a6 6 0 0 0-.4-2.2M6 10c0 4-1.5 5.5-1.5 5.5h12M10 19a2 2 0 0 0 4 0M4 4l16 16",
  "wifi-high": "M4.5 10a11 11 0 0 1 15 0M7.5 13.2a6.5 6.5 0 0 1 9 0M12 17.5h.01",
  "cell-signal-medium": "M4 20h3.5v-5H4zM10 20h3.5V9H10zM16.5 20H20V4h-3.5z",
  clock: "M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18zM12 7v5l3.5 2",
  "calendar-dots": "M4 7.5A1.5 1.5 0 0 1 5.5 6h13A1.5 1.5 0 0 1 20 7.5v11A1.5 1.5 0 0 1 18.5 20h-13A1.5 1.5 0 0 1 4 18.5zM4 10h16M8 4v4M16 4v4M8.5 14h.01M12 14h.01M15.5 14h.01",
  calculator: "M5.5 4.5A1.5 1.5 0 0 1 7 3h10a1.5 1.5 0 0 1 1.5 1.5v15A1.5 1.5 0 0 1 17 21H7a1.5 1.5 0 0 1-1.5-1.5zM8.5 7.5h7M8.5 12h.01M12 12h.01M15.5 12h.01M8.5 16h.01M12 16h.01M15.5 16h.01",
  wallet: "M4 7.5A2.5 2.5 0 0 1 6.5 5H17v3.5M4 7.5v9A2.5 2.5 0 0 0 6.5 19H19v-3.5M20 10.5h-4.5a2 2 0 0 0 0 4H20z",
  "credit-card": "M3 7.5A1.5 1.5 0 0 1 4.5 6h15A1.5 1.5 0 0 1 21 7.5v9a1.5 1.5 0 0 1-1.5 1.5h-15A1.5 1.5 0 0 1 3 16.5zM3 10h18M6.5 14.5h3",
  money: "M2.5 7h19v10h-19zM12 14.5a2.5 2.5 0 1 0 0-5 2.5 2.5 0 0 0 0 5zM6 10.5h.01M18 13.5h.01",
  "piggy-bank": "M3.5 12.5c0-3.6 3.6-6.5 8-6.5 1.3 0 2.5.2 3.6.6L18.5 5v3.4c1 .9 1.6 2 1.8 3.1H22v3.5h-2c-.5 1.4-1.5 2.6-2.8 3.4V20h-3.2v-1.3a11 11 0 0 1-3.5 0V20H7.3v-1.9C5 16.9 3.5 14.9 3.5 12.5zM15.5 12h.01M9.5 6.4C9 4.8 9.6 3.5 9.6 3.5s2 .4 2.7 1.7",
  "currency-btc": "M7 5v14M7 5h6.2a3 3 0 0 1 0 6H7M7 11h7a3.5 3.5 0 0 1 0 7H7M10.5 2.5v3M14 2.5v3M10.5 18v3.5M14 18v3.5",
  receipt: "M5 3.5l2 1.6 2-1.6 2 1.6 2-1.6 2 1.6 2-1.6v17l-2-1.6-2 1.6-2-1.6-2 1.6-2-1.6-2 1.6zM8.5 9.5h7M8.5 14h7",
  "arrows-left-right": "M3 9h18M7 5L3 9l4 4M21 15H3M17 11l4 4-4 4",
  target: "M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18zM12 16.5a4.5 4.5 0 1 0 0-9 4.5 4.5 0 0 0 0 9zM12 13.5a1.5 1.5 0 1 0 0-3 1.5 1.5 0 0 0 0 3z",
  "chart-donut": "M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18zM12 16a4 4 0 1 0 0-8 4 4 0 0 0 0 8zM12 3v5M18.4 16.5l-4.3-2.5",
  "chart-bar": "M4 20h16M7 20v-8M12 20V6M17 20v-5",
  "trend-up": "M4 17l6-6 3.5 3.5L21 7M15 7h6v6",
  "eye-slash": "M4 4l16 16M9.6 9.7a3 3 0 0 0 4.2 4.2M6.4 6.6C3.9 8.2 2.5 10.6 2 12c1.3 3.3 5 7 10 7 1.8 0 3.4-.5 4.8-1.2M9.8 5.3A9.6 9.6 0 0 1 12 5c5 0 8.7 3.7 10 7-.6 1.5-1.7 3.2-3.3 4.6",
  "lock-key": "M6 10.5h12V20H6zM8.5 10.5V7a3.5 3.5 0 0 1 7 0v3.5M12 14a1.5 1.5 0 1 0 0 3 1.5 1.5 0 0 0 0-3zM12 17v1.5",
  "lock-simple": "M5.5 10h13v10h-13zM8.5 10V7a3.5 3.5 0 0 1 7 0v3",
  palette: "M12 20a8 8 0 1 1 0-16c4.4 0 8 3.1 8 7 0 2.2-1.8 3.5-4 3.5h-1.6c-1.2 0-2.2 1-2.2 2.2 0 .4.1.8.3 1.1.2.3.3.7.3 1.1 0 .6-.4 1.1-.8 1.1zM8 9.5h.01M12 7h.01M16 9.5h.01M7.5 14h.01",
  database: "M12 8.5c4.4 0 8-1.2 8-2.75S16.4 3 12 3 4 4.2 4 5.75 7.6 8.5 12 8.5zM4 5.75v12.5C4 19.8 7.6 21 12 21s8-1.2 8-2.75V5.75M4 12c0 1.55 3.6 2.75 8 2.75s8-1.2 8-2.75",
  bell: "M6 10a6 6 0 1 1 12 0c0 4 1.5 5.5 1.5 5.5h-15S6 14 6 10zM10 19a2 2 0 0 0 4 0",
  "bell-ringing": "M6 10a6 6 0 1 1 12 0c0 4 1.5 5.5 1.5 5.5h-15S6 14 6 10zM10 19a2 2 0 0 0 4 0M2.8 8.2a5.5 5.5 0 0 1 2.4-4M21.2 8.2a5.5 5.5 0 0 0-2.4-4",
  lightning: "M13.5 3L5 13.5h6L10.5 21 19 10.5h-6z",
  "arrows-clockwise": "M20 8.5V4M20 8.5h-4.5M4 15.5V20M4 15.5h4.5M19.6 8.5A8 8 0 0 0 5 9.6M4.4 15.5a8 8 0 0 0 14.4-1.1",
  "arrow-clockwise": "M19.5 9.5V4.8M19.5 9.5h-4.7M19.2 9.5A8 8 0 1 0 20 13",
  "folder-plus": "M4 6.5A1.5 1.5 0 0 1 5.5 5h3.6l2 2.5h7.4A1.5 1.5 0 0 1 20 9v9.5a1.5 1.5 0 0 1-1.5 1.5h-13A1.5 1.5 0 0 1 4 18.5zM12 11.5v6M9 14.5h6",
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
  "game-controller": "M8 6.5h8a5.5 5.5 0 0 1 5.4 6.5l-.6 3.4A2.8 2.8 0 0 1 15.4 18L14 16h-4l-1.4 2a2.8 2.8 0 0 1-4.4-1.6L3.6 13A5.5 5.5 0 0 1 8 6.5zM8.5 11h-3M7 9.5v3M15.5 10h.01M17.5 12h.01",
  "device-mobile": "M6.5 3.5h11v17h-11zM10 5.8h4",
} as const;

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
  size?: number;
  color?: string;
  weight?: IconWeight;
  width?: number;
}

const DEFAULT_STROKE = 2.75;

export function Icon({
  name,
  size = 22,
  color = organic.text,
  weight = "regular",
  width,
}: IconProps) {
  const filled = weight === "fill" ? FILLED_PATHS[name] : undefined;
  const stroke = width ?? (weight === "fill" ? 2.3 : DEFAULT_STROKE);
  return (
    <Svg width={size} height={size} viewBox="0 0 24 24" fill="none">
      <Path
        d={filled ?? PATHS[name]}
        fill={filled ? color : "none"}
        stroke={color}
        strokeWidth={filled ? 0 : stroke}
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </Svg>
  );
}

export function isIconName(value: string | null | undefined): value is IconName {
  return value != null && Object.hasOwn(PATHS, value);
}

export const FALLBACK_ICON: IconName = "dots-three";

export function iconOr(value: string | null | undefined, fallback: IconName = FALLBACK_ICON): IconName {
  return isIconName(value) ? value : fallback;
}

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

export function CheckIcon({ size = 13, color = organic.accentFg }: { size?: number; color?: string }) {
  return (
    <Svg width={size} height={size} viewBox="0 0 24 24" fill="none">
      <Path d={PATHS.check} stroke={color} strokeWidth={3.4} strokeLinecap="round" strokeLinejoin="round" />
    </Svg>
  );
}
