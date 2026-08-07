// Metro must watch the monorepo root, or an edit in packages/ui will not hot-reload inside
// an app — the single most reported "shared package is stale" symptom. Every app builds its
// metro.config.js from this factory instead of copy-pasting the resolver settings.
const path = require("node:path");
const { getDefaultConfig } = require("expo/metro-config");

/**
 * @param {string} projectRoot absolute path of the app, i.e. __dirname in the app's config
 * @returns {import('expo/metro-config').MetroConfig}
 */
function createMetroConfig(projectRoot) {
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

  return config;
}

module.exports = { createMetroConfig };
