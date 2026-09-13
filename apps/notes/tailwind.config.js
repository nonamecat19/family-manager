const accent2 = {
  DEFAULT: "#a7a1db",
  100: "#f5f4ff",
  200: "#e7e5fe",
  300: "#d2cefd",
  400: "#b5afe8",
  500: "#9690c9",
  600: "#7972a9",
  700: "#5c5783",
  800: "#423e5d",
  900: "#2b293a",
};

const rail = "#1b1d2c";
const list = "#191b29";

module.exports = {
  content: ["./app/**/*.{ts,tsx}", "./components/**/*.{ts,tsx}", "../../packages/ui/src/**/*.{ts,tsx}"],
  presets: [
    require("nativewind/preset"),
    require("@fm/config/tailwind.preset.cjs"),
    require("@fm/config/nocturne.preset.cjs"),
  ],
  theme: {
    extend: {
      colors: { accent2, rail, list },
      fontFamily: {
        sans: ["Inter_400Regular"],
        med: ["Inter_500Medium"],
        semi: ["Inter_600SemiBold"],
        strong: ["Inter_700Bold"],
      },
    },
  },
};
