import { CheckCircle2 } from "lucide-react"

const orders = [
  { id: "ORD-20240518-001", user: "alice@gmail.com", plan: "标准版 月付", amount: 30, method: "支付宝", date: "2024-05-18 09:15", status: "paid" },
  { id: "ORD-20240518-002", user: "charlie@163.com", plan: "旗舰版 年付", amount: 576, method: "微信支付", date: "2024-05-18 10:42", status: "paid" },
  { id: "ORD-20240517-003", user: "grace@gmail.com", plan: "标准版 月付", amount: 30, method: "支付宝", date: "2024-05-17 14:30", status: "paid" },
  { id: "ORD-20240517-004", user: "henry@yahoo.com", plan: "流量包 100GB", amount: 18, method: "支付宝", date: "2024-05-17 16:55", status: "paid" },
  { id: "ORD-20240516-005", user: "bob@qq.com", plan: "入门版 月付", amount: 15, method: "微信支付", date: "2024-05-16 08:20", status: "paid" },
  { id: "ORD-20240515-006", user: "eve@hotmail.com", plan: "入门版 月付", amount: 15, method: "支付宝", date: "2024-05-15 20:01", status: "refunded" },
]

export default function AdminOrdersPage() {
  const total = orders.filter((o) => o.status === "paid").reduce((s, o) => s + o.amount, 0)

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-foreground">订单管理</h1>
          <p className="text-sm text-muted-foreground mt-1">共 {orders.length} 笔订单 · 总收入 ¥{total}</p>
        </div>
      </div>

      <div className="rounded-xl border border-border bg-card overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="bg-secondary/50 border-b border-border">
                <th className="text-left px-5 py-3 text-muted-foreground font-medium">订单号</th>
                <th className="text-left px-5 py-3 text-muted-foreground font-medium">用户</th>
                <th className="text-left px-5 py-3 text-muted-foreground font-medium">项目</th>
                <th className="text-left px-5 py-3 text-muted-foreground font-medium">金额</th>
                <th className="text-left px-5 py-3 text-muted-foreground font-medium">支付方式</th>
                <th className="text-left px-5 py-3 text-muted-foreground font-medium">时间</th>
                <th className="text-left px-5 py-3 text-muted-foreground font-medium">状态</th>
              </tr>
            </thead>
            <tbody>
              {orders.map((o) => (
                <tr key={o.id} className="border-b border-border/50 hover:bg-accent/50 transition-colors">
                  <td className="px-5 py-3 font-mono text-xs text-muted-foreground">{o.id}</td>
                  <td className="px-5 py-3 text-foreground">{o.user}</td>
                  <td className="px-5 py-3">
                    <span className="text-xs rounded bg-primary/10 text-primary px-2 py-0.5">{o.plan}</span>
                  </td>
                  <td className="px-5 py-3 font-semibold text-foreground">¥{o.amount}</td>
                  <td className="px-5 py-3 text-muted-foreground">{o.method}</td>
                  <td className="px-5 py-3 text-muted-foreground text-xs">{o.date}</td>
                  <td className="px-5 py-3">
                    <span className={`inline-flex items-center gap-1 text-xs rounded-full px-2.5 py-1 ${
                      o.status === "paid"
                        ? "bg-green-500/15 text-green-400"
                        : "bg-yellow-500/15 text-yellow-400"
                    }`}>
                      <CheckCircle2 className="w-3 h-3" />
                      {o.status === "paid" ? "已支付" : "已退款"}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
