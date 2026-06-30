import { useQuery } from '@tanstack/react-query'
import { fetchNovels } from '../api/novels'
import { NovelCard } from '../components/NovelCard'
import { NovelListSkeleton } from '../components/Skeleton'

export function NovelListPage() {
  const { data, isLoading, isError, error } = useQuery({
    queryKey: ['novels'],
    queryFn: fetchNovels,
  })

  if (isLoading) {
    return (
      <div>
        <h1 className="mb-6 text-2xl font-semibold text-white">Danh sách truyện</h1>
        <NovelListSkeleton />
      </div>
    )
  }

  if (isError) {
    return (
      <div className="card border-red-800/50 text-red-300">
        Không tải được danh sách: {(error as Error).message}
      </div>
    )
  }

  const novels = data ?? []

  return (
    <div>
      <h1 className="mb-2 text-2xl font-semibold text-white">Danh sách truyện</h1>
      <p className="mb-6 text-sm text-slate-400">
        Phase 0 — đọc tĩnh từ thư mục <code className="rounded bg-surface-border px-1.5 py-0.5 font-mono text-xs">output/novel</code>
      </p>
      {novels.length === 0 ? (
        <div className="card text-slate-400">Chưa có truyện nào. Chạy ainovel-cli để tạo output/novel.</div>
      ) : (
        <div className="space-y-3">
          {novels.map((novel) => (
            <NovelCard key={novel.slug} novel={novel} />
          ))}
        </div>
      )}
    </div>
  )
}