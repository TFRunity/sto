const { resolve } = require("node:path");
const { getDefaultConfig } = require("expo/metro-config");

const config = getDefaultConfig(__dirname);
config.watchFolders.push(resolve(__dirname, "../common"));

module.exports = config;
