import AdminSidebar from "@/components/admin/AdminSidebar"

export default function AdminLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen bg-background">
      <AdminSidebar />
      <main className="flex-1 overflow-y-auto pt-16 lg:pt-0">
        <div className="mx-auto max-w-6xl px-4 sm:px-6 lg:px-8 py-6 sm:py-8">
          {children}
        </div>
      </main>
    </div>
  )
}
