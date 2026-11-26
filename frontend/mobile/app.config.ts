import type { ConfigContext } from "expo/config";

export default ({ config }: ConfigContext) => {
  const googleServicesFile = process.env.GOOGLE_SERVICES_FILE;
  return googleServicesFile
    ? { ...config, android: { ...config.android, googleServicesFile } }
    : config;
};
