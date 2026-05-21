"use client"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"

type AdminPagerProps = {
  page: number
  pageSize: number
  total: number
  onPageChange: (page: number) => void
  onPageSizeChange?: (pageSize: number) => void
}

const PAGE_SIZES = [10, 12, 20, 50, 100]

export function AdminPager({ page, pageSize, total, onPageChange, onPageSizeChange }: AdminPagerProps) {
  const totalPages = Math.max(1, Math.ceil(total / pageSize))
  const currentPage = Math.min(Math.max(page, 1), totalPages)
  const start = total === 0 ? 0 : (page - 1) * pageSize + 1
  const end = Math.min(total, page * pageSize)

  const jumpTo = (value: string) => {
    const next = Number(value)
    if (!Number.isFinite(next)) return
    onPageChange(Math.min(Math.max(1, Math.trunc(next)), totalPages))
  }

  return (
    <div className="flex flex-col gap-3 pt-4 text-sm text-muted-foreground lg:flex-row lg:items-center lg:justify-between">
      <span>
        共 {total} 条，当前 {start}-{end}
      </span>
      <div className="flex flex-wrap items-center gap-2">
        {onPageSizeChange && (
          <Select value={String(pageSize)} onValueChange={(v) => onPageSizeChange(Number(v))}>
            <SelectTrigger className="h-9 w-[110px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {PAGE_SIZES.map((size) => (
                <SelectItem key={size} value={String(size)}>每页 {size}</SelectItem>
              ))}
            </SelectContent>
          </Select>
        )}
        <Button
          type="button"
          size="sm"
          variant="outline"
          onClick={() => onPageChange(page - 1)}
          disabled={page <= 1}
        >
          上一页
        </Button>
        <span className="min-w-20 text-center">
          {currentPage} / {totalPages}
        </span>
        <Button
          type="button"
          size="sm"
          variant="outline"
          onClick={() => onPageChange(page + 1)}
          disabled={page >= totalPages}
        >
          下一页
        </Button>
        <div className="flex items-center gap-1">
          <span>跳至</span>
          <Input
            key={currentPage}
            type="number"
            min="1"
            max={totalPages}
            defaultValue={currentPage}
            onKeyDown={(e) => {
              if (e.key === "Enter") jumpTo((e.target as HTMLInputElement).value)
            }}
            onBlur={(e) => jumpTo(e.target.value)}
            className="h-9 w-16 text-center"
          />
          <span>页</span>
        </div>
      </div>
    </div>
  )
}
