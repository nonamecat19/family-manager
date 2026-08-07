// Tokens come from the shared preset; this file only says where classes are used.
/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./app/**/*.{ts,tsx}", "./components/**/*.{ts,tsx}", "../../packages/ui/src/**/*.{ts,tsx}"],
  presets: [require("nativewind/preset"), require("@fm/config/tailwind.preset.cjs")],
};
