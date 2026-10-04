const ENDPOINTS = {
  production: {
    auth: "https://auth.nonamecat.pp.ua",
    family: "https://family.nonamecat.pp.ua",
    recipes: "https://recipes.nonamecat.pp.ua",
    notifications: "https://notifications.nonamecat.pp.ua",
  },
  local: {
    auth: "http://localhost:8081",
    family: "http://localhost:8082",
    recipes: "http://localhost:8084",
    notifications: "http://localhost:8087",
  },
  emulator: {
    auth: "http://10.0.2.2:8081",
    family: "http://10.0.2.2:8082",
    recipes: "http://10.0.2.2:8084",
    notifications: "http://10.0.2.2:8087",
  },
};

const env = process.env.EXPO_PUBLIC_API_ENV ?? "production";
const endpoints = ENDPOINTS[env];
if (!endpoints) {
  throw new Error(
    `EXPO_PUBLIC_API_ENV must be one of ${Object.keys(ENDPOINTS).join(", ")}, got "${env}". ` +
      `A typo here would otherwise silently fall back to a backend you did not mean to use.`,
  );
}

const authUrl = process.env.EXPO_PUBLIC_AUTH_URL ?? endpoints.auth;
const familyUrl = process.env.EXPO_PUBLIC_FAMILY_URL ?? endpoints.family;
const notificationsUrl = process.env.EXPO_PUBLIC_NOTIFICATIONS_URL ?? endpoints.notifications;
const recipesUrl = process.env.EXPO_PUBLIC_RECIPES_URL ?? endpoints.recipes;
const apiBaseUrl = process.env.EXPO_PUBLIC_API_BASE_URL ?? recipesUrl;

module.exports = {
  expo: {
    name: "Family Recipes",
    slug: "fm-recipes",
    scheme: "fmrecipes",
    version: "0.1.0",
    orientation: "portrait",
    icon: "./assets/icon.png",
    userInterfaceStyle: "automatic",
    newArchEnabled: true,
    plugins: [
      ...(env === "production" ? [] : [require("@fm/config/cleartext.plugin.cjs")]),
      "expo-router",
      "expo-secure-store",
      "expo-localization",
      "expo-notifications",
      [
        "expo-image-picker",
        {
          photosPermission: "Family Recipes uses your photo library to attach a photo to a recipe.",
        },
      ],
    ],
    experiments: {
      typedRoutes: true,
    },
    extra: {
      "//": "Endpoints come from EXPO_PUBLIC_API_ENV (default: production). See the top of this file.",
      apiEnv: env,
      telegramBot: process.env.EXPO_PUBLIC_TELEGRAM_BOT ?? "",
      apiBaseUrl,
      serviceUrls: {
        auth: authUrl,
        family: familyUrl,
        notifications: notificationsUrl,
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
        foregroundImage: "./assets/adaptive-icon.png",
        backgroundColor: "#C4643C",
      },
    },
    web: {
      bundler: "metro",
      output: "static",
      favicon: "./assets/favicon.png",
    },
  },
};
