import { NavLink, Outlet } from 'react-router-dom'
import { BookMarked, LayoutDashboard } from 'lucide-react'
import { useQuery } from '@tanstack/react-query'
import { fetchNovels } from '../api/novels'

export function Layout() {
  const { data: novels = [] } = useQuery({
    queryKey: ['novels'],
    queryFn: fetchNovels,
  })

  return (
    <div className="flex min-h-screen">
      <aside className="flex w-64 shrink-0 flex-col border-r border-surface-border bg-surface-raised">
        <div className="border-b border-surface-border px-4 py-5">
          <div className="flex items-center gap-2 text-white">
            <LayoutDashboard size={20} className="text-accent" />
            <span className="font-semibold tracking-tight">ainovel</span>
          </div>
          <p className="mt-1 text-xs text-slate-500">Web Dashboard</p>
        </div>
        <nav className="flex-1 overflow-y-auto p-3">
          <NavLink
            to="/novels"
            end
            className={({ isActive }) =>
              `mb-1 flex items-center gap-2 rounded-lg px-3 py-2 text-sm transition ${
                isActive ? 'bg-accent/20 text-white' : 'text-slate-400 hover:bg-surface-border/50 hover:text-slate-200'
              }`
            }
          >
            <BookMarked size={16} />
            Tất cả truyện
          </NavLink>
          <div className="mt-4 px-3 text-xs font-medium uppercase tracking-wider text-slate-500">
            Truyện ({novels.length})
          </div>
          <ul className="mt-2 space-y-0.5">
            {novels.map((n) => (
              <li key={n.slug}>
                <NavLink
                  to={`/novels/${n.slug}`}
                  className={({ isActive }) =>
                    `block truncate rounded-lg px-3 py-2 text-sm transition ${
                      isActive ? 'bg-surface-border text-white' : 'text-slate-400 hover:text-slate-200'
                    }`
                  }
                >
                  {n.title}
                </NavLink>
              </li>
            ))}
          </ul>
        </nav>
      </aside>
      <main className="flex-1 overflow-y-auto p-6 lg:p-8">
        <Outlet />
      </main>
    </div>
  )
}