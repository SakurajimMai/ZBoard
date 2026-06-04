import assert from "node:assert/strict"
import { readFileSync } from "node:fs"
import path from "node:path"
import test from "node:test"
import { fileURLToPath } from "node:url"

const root = path.join(path.dirname(fileURLToPath(import.meta.url)), "..")

function readSource(...parts) {
  return readFileSync(path.join(root, ...parts), "utf8")
}

test("后台概览流量图不能固定使用 TB 尺度", () => {
  const source = readSource("components", "admin", "AdminOverview.tsx")

  assert.ok(source.includes("selectTrafficUnit"), "后台概览应按实际流量动态选择 B/KB/MB/GB/TB 单位")
  assert.ok(source.includes("formatTrafficValue"), "后台概览应使用动态单位格式化坐标与提示")
  assert.ok(!source.includes("近 7 天总流量 (TB)"), "标题不应固定显示 TB")
  assert.ok(!source.includes("Math.max(...data.map((d) => d.tb), 1)"), "纵轴最大值不应固定兜底到 1 TB")
  assert.ok(!source.includes("d.tb / max"), "点位计算不应继续按 TB 比例缩放")
})

test("用户每日流量图不能固定使用 GB 尺度", () => {
  const source = readSource("components", "dashboard", "Subscription.tsx")

  assert.ok(source.includes("selectTrafficUnit"), "用户每日流量图应按实际流量动态选择单位")
  assert.ok(source.includes("formatTrafficValue"), "用户每日流量图应使用动态单位格式化提示")
  assert.ok(!source.includes('unit=" GB"'), "用户每日流量图纵轴不应固定 GB")
  assert.ok(!source.includes("bytesToGB(d.upload)"), "每日上传流量不应固定转换为 GB")
  assert.ok(!source.includes("bytesToGB(d.download)"), "每日下载流量不应固定转换为 GB")
})
