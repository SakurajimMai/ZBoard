import assert from "node:assert/strict"
import { existsSync, readFileSync } from "node:fs"
import path from "node:path"
import test from "node:test"
import { fileURLToPath } from "node:url"

const root = path.join(path.dirname(fileURLToPath(import.meta.url)), "..")

function readSource(...parts) {
  return readFileSync(path.join(root, ...parts), "utf8")
}

test("管理端侧边栏在用户管理下面提供活跃用户入口", () => {
  const source = readSource("components", "admin", "AdminSidebar.tsx")
  const usersIndex = source.indexOf('href: "/admin/users"')
  const activeIndex = source.indexOf('href: "/admin/active-users"')

  assert.ok(usersIndex >= 0, "侧边栏缺少用户管理入口")
  assert.ok(activeIndex >= 0, "侧边栏缺少活跃用户入口")
  assert.ok(activeIndex > usersIndex, "活跃用户入口应位于用户管理后面")
  assert.ok(source.includes("Activity"), "活跃用户入口应使用活动状态图标")
})

test("管理端活跃用户页面和 API 封装存在", () => {
  assert.ok(existsSync(path.join(root, "app", "admin", "active-users", "page.tsx")), "缺少 /admin/active-users 页面")

  const apiSource = readSource("lib", "api.ts")
  assert.ok(apiSource.includes("adminGetActiveUsers"), "缺少 adminGetActiveUsers API 封装")
  assert.ok(apiSource.includes("/api/admin/v1/active-users"), "API 封装应请求活跃用户接口")

  const pageSource = readSource("components", "admin", "AdminActiveUsers.tsx")
  assert.ok(pageSource.includes("adminGetActiveUsers"), "活跃用户组件应调用 adminGetActiveUsers")
  assert.ok(pageSource.includes("lastSevenDays"), "页面应提供最近 7 天日期筛选")
  assert.ok(pageSource.includes("活跃用户"), "页面应显示活跃用户标题")
})
