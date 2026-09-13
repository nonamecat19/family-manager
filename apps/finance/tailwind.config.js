module.exports = {
  content: ["./app/**/*.{ts,tsx}", "./components/**/*.{ts,tsx}", "../../packages/ui/src/**/*.{ts,tsx}"],
  presets: [
    require("nativewind/preset"),
    require("@fm/config/tailwind.preset.cjs"),
    require("@fm/config/nocturne.preset.cjs"),
  ],
  theme: {
    extend: {
      colors: {
        canvas: "#101120",
      },
      spacing: {
        n1: "2.8px",
        n2: "5.6px",
        n3: "8.4px",
        n4: "11.2px",
        n5: "16.8px",
        n6: "22.4px",
      },
    },
  },
};
