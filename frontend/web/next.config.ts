import { resolve } from "node:path";
import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  agentRules: false,
  output: "standalone",
  outputFileTracingRoot: resolve(__dirname, ".."),
  turbopack: { root: resolve(__dirname, "..") },
  async rewrites() {
    if (process.env.NODE_ENV !== "development") return [];
    const nginxOrigin = process.env.STO_NGINX_ORIGIN ?? "http://127.0.0.1";
    return [{ source: "/api/:path*", destination: `${nginxOrigin}/api/:path*` }];
  },
};

export default nextConfig;
