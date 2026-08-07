// Shared Babel preset for every Expo app. NativeWind's jsxImportSource must be applied here,
// not per app, or className props silently become no-ops in one app and work in another.
module.exports = function createBabelConfig(api) {
  api.cache(true);
  return {
    presets: [
      ["babel-preset-expo", { jsxImportSource: "nativewind" }],
      "nativewind/babel",
    ],
  };
};
