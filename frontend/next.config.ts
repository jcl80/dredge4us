import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  async redirects() {
    return [{ source: "/boards", destination: "/coverage", permanent: false }];
  },
};

export default nextConfig;
