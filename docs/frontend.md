# Zboard 前端

Zboard 前端是一个独立的 Next.js 16 工程，**不归 Go 后端管**。后端只暴露 JSON API，前端独立部署、独立调用。本文档面向前端开发者：技术栈、路由结构、组件组织、API 客户端、国际化、主题系统与跨切面基础设施。

> 源码位置：`frontend/`。后端 API 契约见 [API 设计](./api.md)，整体架构见 [系统架构](./architecture.md)。

---

## 1. 技术栈

| 组件 | 技术 | 版本 |
|------|------|------|
| 框架 | Next.js（App Router + Turbopack） | 16.2.6 |
| UI 库 | React | 19 |
| 样式 | Tailwind CSS（CSS-first 配置） | 4.2.0 |
| 组件原语 | shadcn/ui（基于 Radix UI） | — |
| 图标 | lucide-react | 0.564.0 |
| 主题 | next-themes | 0.4.6 |
| 轻提示 | sonner | 1.7.1 |
| 图表 | recharts | 2.15.0 |
| 包管理 | pnpm | — |

要点：

- **Tailwind 4 是 CSS-first 配置**：没有 `tailwind.config.js`，所有 token 和主题定义在 `app/globals.css` 的 `@theme` 块里，PostCSS 插件见 `postcss.config.mjs`（`@tailwindcss/postcss`）。
- **TypeScript 构建错误被放行**：`next.config.mjs` 里 `typescript.ignoreBuildErrors: true`，所以 `pnpm build` 不会因类型错误失败。**务必在提交前单独跑 `npx tsc --noEmit` 把关。**
- **standalone 输出**：`output: 'standalone'`，用于精简 Docker 镜像。
- `components.json` 是 shadcn CLI 的配置（别名、样式风格）。

### 命令

```bash
cd frontend
pnpm install
pnpm dev      # 开发服务器（默认 :3000，容器内映射到宿主 :3001）
pnpm build    # 生产构建（standalone）
pnpm start    # 跑构建产物
pnpm lint     # eslint
npx tsc --noEmit   # 类型检查（构建不做,必须手动跑）
```

---

## 2. 路由结构（App Router）

```
app/
├── layout.tsx              根布局:字体 / 主题 / i18n / Toaster / ConfirmProvider
├── globals.css             设计 token + 全局样式
├── page.tsx                落地页(Navbar + Hero + Features + Nodes + Pricing + FAQ + Footer)
├── login/                  用户登录
├── register/               用户注册(支持邮箱验证码 + 验证码人机校验)
├── forgot-password/        找回密码
├── dashboard/              用户端(需登录)
│   ├── layout.tsx          鉴权 + Sidebar + AnnouncementPopup
│   ├── page.tsx            概览(订阅链接 / 流量 / 套餐购买)
│   ├── subscription/       订阅详情 + 流量图表
│   ├── knowledge/          知识库(用户视角)
│   ├── ticket/             工单
│   └── settings/           账号设置(改密 / 注销)
└── admin/                  管理后台(需管理员登录)
    ├── layout.tsx          鉴权 + AdminSidebar
    ├── login/              管理员登录(无鉴权守卫)
    ├── page.tsx            数据概览
    ├── users/              用户管理
    ├── nodes/              节点管理
    ├── plans/              套餐管理
    ├── orders/             订单管理
    ├── announcements/      公告管理
    ├── knowledge/          知识库管理
    ├── tickets/            工单管理
    └── settings/           系统设置(站点 / 支付 / SMTP / 验证码)
```

### 三个布局与鉴权

| 布局 | 鉴权方式 | 说明 |
|------|----------|------|
| `app/layout.tsx` | 无 | 注入字体、`ThemeProvider`、`I18nProvider`、`ConfirmProvider`、`Toaster`。服务端读 cookie/Accept-Language 解析初始 locale |
| `app/dashboard/layout.tsx` | 客户端 | `getToken()` 无 token → 跳 `/login`；`getMe()` 失败 → 跳 `/login`。校验通过才渲染 `Sidebar` |
| `app/admin/layout.tsx` | 客户端 | `getAdminToken()` + `adminGetMe()`。`/admin/login` 自身跳过守卫 |

鉴权是**客户端守卫**（localStorage token + 拉取 `/me` 验证），不是服务端中间件。真正的访问控制在后端 —— 前端守卫只是体验层，绕过它也拿不到数据。

---

## 3. 组件组织

```
components/
├── ui/            57 个 shadcn/ui 原语(button, card, dialog, alert-dialog,
│                  select, table, sheet, sonner, ...);主题感知,勿手改单个颜色
├── admin/         AdminSidebar, AdminOverview, AdminUsers, AdminNodes,
│                  AdminPlans, AdminAnnouncements, AdminKnowledge, AdminPager
├── dashboard/     Sidebar, Overview, Subscription, QRCodeDialog, AnnouncementPopup
├── landing/       Navbar, Hero, Features, Nodes, Pricing, FAQ, Footer
├── theme-provider.tsx   next-themes 封装
├── ThemeToggle.tsx      亮/暗/跟随系统三态切换
├── LanguageSwitcher.tsx 语言切换(支持 iconOnly 纯图标模式)
├── confirm-dialog.tsx   ConfirmProvider + useConfirm hook
├── captcha.tsx          Turnstile / reCAPTCHA / hCaptcha 统一封装
└── MarkdownRenderer.tsx 知识库 Markdown 渲染
```

约定：

- **图标一律用 lucide-react SVG，禁用 emoji 当图标。** 描边宽度统一。
- shadcn 原语已接主题 token，不要在调用处写死 `bg-blue-500` 之类的颜色——用语义 token（`bg-primary`、`text-muted-foreground`、`border-border`）。
- admin / dashboard 侧边栏都有移动端抽屉 + 桌面固定栏两套渲染。

---

## 4. API 客户端层（`lib/api.ts`）

所有后端调用都过这一层。

### Base URL 解析

```ts
getApiBase()  // NEXT_PUBLIC_API_URL 去尾斜杠;为空则返回 ""(同源)
```

- **留空（推荐）**：走同源 `/api/*`，由 `next.config.mjs` 的 `rewrites` 反代到 `API_INTERNAL_URL`（容器内 `http://api:3000`）。CDN 只需代理 80/443，不暴露 3000。
- **显式设 `NEXT_PUBLIC_API_URL`**：用于独立 API 子域名（如 `https://api.example.com`），需在后端 `ZBOARD_CORS_ORIGINS` 加上前端域名。

### 鉴权 token

- 用户 token 存 `localStorage["zboard_token"]`；管理员存 `localStorage["zboard_admin_token"]`。
- `request<T>()`（用户）和 `adminRequest<T>()`（管理员）两个 helper 自动附 `Authorization: Bearer <token>`。
- 错误统一抛 `ApiError`（带 `status` / `code` / `message`），UI 层 `catch` 后用 toast 展示。

### 订阅 URL 构造

```ts
getSubscriptionBase(settings)   // 优先 subscription_domain → NEXT_PUBLIC_API_URL → window.origin
buildSubscriptionUrl(token, target?, settings?)        // 主链接
buildSubscriptionUrlFromBase(base, token, target?)     // 备用域名链接
```

主/备用订阅域名由后台「站点设置」配置（`subscription_domain` / `backup_subscription_domain`），指向**控制面**而非节点。详见 [订阅系统](./subscription.md)。

---

## 5. 国际化（`lib/i18n/`）

```
lib/i18n/
├── locales.ts    8 种语言定义(code / 原生名 / dir)
├── resolve.ts    locale 检测:cookie → Accept-Language → 默认 en
├── context.tsx   I18nProvider + useI18n() hook
├── auth.ts       登录/注册/找回密码文案
└── dashboard.ts  用户端 + 管理后台文案
```

- **支持语言**：`zh-CN`、`zh-TW`、`en`、`fa`、`ja`、`vi`、`ko`、`ru`。默认 `en`。`fa` 是 RTL。
- 服务端在 `app/layout.tsx` 读 `LOCALE_COOKIE_KEY` cookie 与 `Accept-Language` 头解析初始 locale，避免首屏闪烁，并设到 `<html lang dir>`。
- 客户端用 `useI18n()` 取 `{ t, locale, setLocale }`。新增文案需在 `auth.ts` / `dashboard.ts` 的**所有 8 种语言**里补 key —— 类型系统会强制（缺 key 则 `tsc` 报错）。

---

## 6. 主题系统（`app/globals.css` + next-themes）

### Token 体系

- **OKLCH 颜色**，蓝/靛主色（`--primary`），亮/暗两套完整定义。
- 语义 token：`--background` / `--foreground` / `--card` / `--primary` / `--muted` / `--destructive` / `--success` / `--border` / `--ring` / `--sidebar-*` / `--chart-1..5`。
- **分层阴影体系**：`--shadow-xs/sm/md/lg`（冷蓝灰色温，按 elevation 递进），亮暗各一套。`.card-shadow` / `.card-shadow-hover` 引用它们。
- 圆角统一 `--radius: 0.75rem`，派生 `--radius-sm/md/lg/xl`。

### 字体

- 正文：**Inter** + Noto Sans SC（`--font-sans`）
- 标题展示：**Space Grotesk**（`--font-display` + `.font-display` 工具类，字距收紧）
- 通过 `next/font` 在 `app/layout.tsx` 注入（`display: 'swap'`），不阻塞渲染。

### 深色模式

- `next-themes` + `components/theme-provider.tsx`，`app/layout.tsx` 配置 `attribute="class" defaultTheme="system" enableSystem`。
- `components/ThemeToggle.tsx`：亮 / 暗 / 跟随系统三态循环，图标反映当前生效主题，hydration 安全（挂载前渲染占位）。已接入两个侧边栏 + 落地页 Navbar。
- 全局 `prefers-reduced-motion` 降级：base 层关闭非必要动画。

---

## 7. 跨切面基础设施

### Toast（轻提示）

- `components/ui/sonner.tsx` 是主题感知的 `Toaster`，在 `app/layout.tsx` 挂载（`position="top-center" richColors closeButton`）。
- 任意组件 `import { toast } from "sonner"` 后调 `toast.success()` / `toast.error()`。
- **禁用原生 `alert()`** —— 全部走 toast。

### 确认对话框

- `components/confirm-dialog.tsx` 提供 Promise 化的 `useConfirm()` hook，基于 shadcn `AlertDialog`。
- 用法替代原生 `confirm()`：

```ts
const confirm = useConfirm()
if (!(await confirm({ title: "删除节点", description: "...", destructive: true }))) return
```

- `destructive: true` 时确认按钮用红色配色。`<ConfirmProvider>` 在 `app/layout.tsx` 包住全站。

### 安全响应头与反代（`next.config.mjs`）

- **CSP**：`script-src` / `connect-src` / `frame-src` 放行同源 + Cloudflare beacon + 三家验证码域名（Turnstile / reCAPTCHA / hCaptcha）+ `NEXT_PUBLIC_API_URL` origin；`img-src` 含 `https:`（知识库外链图）。
- 其它头：`X-Frame-Options: DENY`、`X-Content-Type-Options: nosniff`、`Referrer-Policy`、`Permissions-Policy`。
- **rewrites**：同源 `/api/:path*` 和 `/health` 反代到 `API_INTERNAL_URL`（默认 `http://api:3000`）。
- **viewport**：`app/layout.tsx` 导出 `viewport`（`width=device-width, initialScale=1` + `themeColor` 亮暗双色），保证移动端按真实宽度渲染。

> CSP 是构建期生成的，改 `next.config.mjs` 后**必须重新构建并重启前端**才生效。

---

## 8. 环境变量

| 变量 | 作用 |
|------|------|
| `NEXT_PUBLIC_API_URL` | 浏览器端 API 基址。留空走同源反代（推荐）；设为独立 API 域名时需后端 CORS 放行 |
| `API_INTERNAL_URL` | 服务端反代目标，容器内默认 `http://api:3000` |
| `NEXT_PUBLIC_VERCEL_ANALYTICS` | `1`/`true` 启用 Vercel Analytics |

部署见 [部署方案](./deploy.md)。

---

## 9. 开发约定

- **先跑 `npx tsc --noEmit`**：构建放行类型错误，类型检查必须手动做。
- **用语义 token，不写死颜色**：保证亮暗主题与品牌一致性。
- **反馈用 toast / useConfirm，不用原生弹窗。**
- **新增文案补齐 8 种语言。**
- **图标用 lucide SVG，不用 emoji。**
- 触摸目标 ≥44px（输入框 `h-11 md:h-9`），尊重移动端与无障碍。
- 数据列（流量、价格、到期时间）用 `.tabular-nums` 等宽数字，刷新不跳动。
