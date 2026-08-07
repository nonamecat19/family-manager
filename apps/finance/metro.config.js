// Watches the workspace root so edits in packages/* hot-reload here. See @fm/config.
const { createMetroConfig } = require("@fm/config/metro.config.cjs");

module.exports = createMetroConfig(__dirname);
