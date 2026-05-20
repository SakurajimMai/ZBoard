/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'standalone',
  typescript: {
    ignoreBuildErrors: true,
  },
  images: {
    unoptimized: true,
  },
  // 把同源 /api/* 反代到后端容器,让前端走 CDN 时不必暴露 :3000
  // 容器内通过 docker-compose 服务名 `api` 互通;裸机部署可设 API_INTERNAL_URL 覆盖
  async rewrites() {
    const target = process.env.API_INTERNAL_URL || 'http://api:3000'
    return [
      { source: '/api/:path*', destination: `${target}/api/:path*` },
      { source: '/health', destination: `${target}/health` },
    ]
  },
}

export default nextConfig
