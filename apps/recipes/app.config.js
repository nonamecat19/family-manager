// Dynamic config so `apiBaseUrl`/`serviceUrls` can point at a LAN IP for real-device
// testing (localhost on a physical device resolves to the device itself, not the dev
// machine). Emulator/simulator default to localhost as before.
const apiBaseUrl = process.env.EXPO_PUBLIC_API_BASE_URL ?? "http://localhost:8084";
const authUrl = process.env.EXPO_PUBLIC_AUTH_URL ?? "http://localhost:8081";
const familyUrl = process.env.EXPO_PUBLIC_FAMILY_URL ?? "http://localhost:8082";
const recipesUrl = process.env.EXPO_PUBLIC_RECIPES_URL ?? apiBaseUrl;

module.exports = {
  expo: {
    name: "Family Recipes",
    slug: "fm-recipes",
    scheme: "fmrecipes",
    version: "0.1.0",
    orientation: "portrait",
    userInterfaceStyle: "automatic",
    newArchEnabled: true,
    plugins: ["expo-router", "expo-secure-store"],
    experiments: {
      typedRoutes: true,
    },
    extra: {
      "//": "Development points at each service directly; deployed, apiBaseUrl is the gateway and serviceUrls is removed. Override via EXPO_PUBLIC_API_BASE_URL etc. for real-device testing (use your machine's LAN IP, not localhost).",
      apiBaseUrl,
      serviceUrls: {
        auth: authUrl,
        family: familyUrl,
        recipes: recipesUrl,
      },
    },
    ios: {
      supportsTablet: true,
      bundleIdentifier: "dev.familymanager.recipes",
    },
    android: {
      package: "dev.familymanager.recipes",
      adaptiveIcon: {
        backgroundColor: "#C05621",
      },
      // Backend runs over plain HTTP in dev/LAN testing; without this, release builds
      // (and any build without the debug manifest override) block cleartext requests.
      usesCleartextTraffic: true,
    },
    web: {
      bundler: "metro",
      output: "static",
    },
  },
};
