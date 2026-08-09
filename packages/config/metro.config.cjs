// Metro must watch the monorepo root, or an edit in packages/ui will not hot-reload inside
// an app — the single most reported "shared package is stale" symptom. Every app builds its
// metro.config.js from this factory instead of copy-pasting the resolver settings.
const path = require("node:path");
const { getDefaultConfig } = require("expo/metro-config");
const { withNativeWind } = require("nativewind/metro");

/**
 * @param {string} projectRoot absolute path of the app, i.e. __dirname in the app's config
 * @param {string} cssEntry app-relative path to the Tailwind entry file
 * @returns {import('expo/metro-config').MetroConfig}
 */
function createMetroConfig(projectRoot, cssEntry = "./global.css") {
  const workspaceRoot = path.resolve(projectRoot, "../..");
  const config = getDefaultConfig(projectRoot);

  // Watch the whole workspace so shared packages trigger rebuilds.
  config.watchFolders = [workspaceRoot];

  // Resolve from the app first, then the hoisted root store. pnpm's symlinked layout needs
  // both, and nodeModulesPaths order decides which wins.
  config.resolver.nodeModulesPaths = [
    path.resolve(projectRoot, "node_modules"),
    path.resolve(workspaceRoot, "node_modules"),
  ];

  // pnpm symlinks packages; without this Metro resolves through the link target and loads
  // React twice.
  config.resolver.unstable_enableSymlinks = true;
  config.resolver.disableHierarchicalLookup = true;

  // Respect the "exports" field in workspace package.json files so wildcard
  // exports (e.g. @fm/sdk's "./*": "./src/*.ts") resolve correctly.
  config.resolver.unstable_enablePackageExports = true;

  return withNativeWind(config, { input: cssEntry });
}

module.exports = { createMetroConfig };
