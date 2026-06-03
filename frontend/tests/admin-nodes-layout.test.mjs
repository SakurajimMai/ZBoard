import assert from "node:assert/strict"
import { readFileSync } from "node:fs"
import path from "node:path"
import test from "node:test"
import { fileURLToPath } from "node:url"

const sourcePath = path.join(
  path.dirname(fileURLToPath(import.meta.url)),
  "..",
  "components",
  "admin",
  "AdminNodes.tsx",
)

const source = readFileSync(sourcePath, "utf8")

function classNamesFor(pattern) {
  const match = source.match(pattern)
  assert.ok(match, "没有找到目标 className")
  return new Set(match[1].split(/\s+/))
}

function assertHasClasses(classes, expected) {
  for (const className of expected) {
    assert.ok(classes.has(className), `缺少 className: ${className}`)
  }
}

test("桌面节点列表保留横向滚动容器", () => {
  assert.ok(source.includes('className="overflow-x-auto"'), "节点列表缺少横向滚动容器")
  assert.ok(source.includes('className="xl:min-w-[84rem]"'), "桌面节点列表缺少超宽内容兜底")
  assert.ok(
    !source.includes('className="rounded-lg border bg-card overflow-hidden"'),
    "节点列表外层不应继续吞掉横向滚动",
  )
})

test("节点编辑弹窗把滚动交给内容区而不是硬编码高度", () => {
  const dialogClasses = classNamesFor(/<DialogContent className="([^"]+)"/)
  assertHasClasses(dialogClasses, ["flex", "max-h-[90dvh]", "flex-col", "overflow-hidden"])
  assert.ok(
    source.includes('className="min-h-0 flex-1 overflow-y-auto px-4 py-5 space-y-5 sm:px-6"'),
    "表单内容区应成为唯一的纵向滚动区域",
  )
  assert.ok(
    !source.includes("max-h-[calc(92dvh-5rem)]") && !source.includes("max-h-[calc(92dvh-10rem)]"),
    "弹窗内容区不应继续依赖硬编码 calc 高度",
  )
})
