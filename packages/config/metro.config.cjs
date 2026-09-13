const path = require("node:path");
const { getDefaultConfig } = require("expo/metro-config");
const { withNativeWind } = require("nativewind/metro");

function createMetroConfig(projectRoot, cssEntry = "./global.css") {
  const workspaceRoot = path.resolve(projectRoot, "../..");
  const config = getDefaultConfig(projectRoot);

  config.watchFolders = [workspaceRoot];

  config.resolver.nodeModulesPaths = [
    path.resolve(projectRoot, "node_modules"),
    path.resolve(workspaceRoot, "node_modules"),
  ];

  config.resolver.unstable_enableSymlinks = true;
  config.resolver.disableHierarchicalLookup = true;

  config.resolver.unstable_enablePackageExports = true;

  return withNativeWind(config, { input: cssEntry });
}

module.exports = { createMetroConfig };
