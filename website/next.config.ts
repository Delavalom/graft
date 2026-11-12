import { createMDX } from "fumadocs-mdx/next";
import type { NextConfig } from "next";

// Static export: served by Cloudflare Workers Static Assets (see wrangler.jsonc).
const nextConfig: NextConfig = {
  reactStrictMode: true,
  output: "export",
  images: { unoptimized: true },
};

const withMDX = createMDX();

export default withMDX(nextConfig);
