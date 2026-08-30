/** @type {import('next').NextConfig} */
const nextConfig = {
  images: {
    remotePatterns: [
      {
        // CloudFront domain for processed media — set NEXT_PUBLIC_CLOUDFRONT_URL env var
        protocol: "https",
        hostname: "*.cloudfront.net",
      },
    ],
  },
};

module.exports = nextConfig;
