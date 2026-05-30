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
  // 安全响应头:防点击劫持、MIME 嗅探、Referrer 泄露,并约束资源加载来源。
  // CSP 允许 'unsafe-inline' 样式(shadcn/ui 注入内联样式,Next 也会内联关键 CSS),
  // 脚本限制在同源 + 'unsafe-inline'(Next 运行时需要)。
  //
  // connect-src / script-src / frame-src 还需放行两类来源,否则已有配置场景会被
  // 浏览器直接拦截:
  //   1. 独立 API 域名:设置 NEXT_PUBLIC_API_URL 时前端直连该 origin(不走同源反代)。
  //   2. 人机验证:启用 Turnstile / reCAPTCHA / hCaptcha 时需加载其脚本与 iframe。
  async headers() {
    const apiOrigin = (() => {
      const raw = (process.env.NEXT_PUBLIC_API_URL || '').trim().replace(/\/+$/, '')
      if (!raw) return ''
      try {
        return new URL(raw).origin
      } catch {
        return ''
      }
    })()

    // 三家验证码服务的官方域名。一次性全部放行,避免管理员切换 provider 后
    // 又要改部署配置;这些都是受信任的固定来源,放行成本可忽略。
    const captchaScript = [
      'https://challenges.cloudflare.com',
      'https://www.google.com',
      'https://www.gstatic.com',
      'https://www.recaptcha.net',
      'https://hcaptcha.com',
      'https://*.hcaptcha.com',
    ]
    const captchaFrame = [
      'https://challenges.cloudflare.com',
      'https://www.google.com',
      'https://recaptcha.net',
      'https://www.recaptcha.net',
      'https://hcaptcha.com',
      'https://*.hcaptcha.com',
    ]
    const captchaConnect = [
      'https://challenges.cloudflare.com',
      'https://www.google.com',
      'https://www.recaptcha.net',
      'https://hcaptcha.com',
      'https://*.hcaptcha.com',
    ]

    const connectSrc = ["'self'", apiOrigin, ...captchaConnect].filter(Boolean)
    const scriptSrc = ["'self'", "'unsafe-inline'", "'unsafe-eval'", ...captchaScript]
    const frameSrc = ["'self'", ...captchaFrame]

    const csp = [
      "default-src 'self'",
      `script-src ${scriptSrc.join(' ')}`,
      "style-src 'self' 'unsafe-inline'",
      "img-src 'self' data: blob:",
      "font-src 'self' data:",
      `connect-src ${connectSrc.join(' ')}`,
      `frame-src ${frameSrc.join(' ')}`,
      "frame-ancestors 'none'",
      "base-uri 'self'",
      "form-action 'self'",
      "object-src 'none'",
    ].join('; ')
    return [
      {
        source: '/:path*',
        headers: [
          { key: 'Content-Security-Policy', value: csp },
          { key: 'X-Frame-Options', value: 'DENY' },
          { key: 'X-Content-Type-Options', value: 'nosniff' },
          { key: 'Referrer-Policy', value: 'strict-origin-when-cross-origin' },
          { key: 'X-DNS-Prefetch-Control', value: 'off' },
          { key: 'Permissions-Policy', value: 'camera=(), microphone=(), geolocation=()' },
        ],
      },
    ]
  },
}

export default nextConfig
