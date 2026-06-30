import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import {
  fetchCharacters,
  fetchNovel,
  fetchOutline,
  fetchPremise,
} from '../api/novels'
import { DetailSkeleton } from '../components/Skeleton'
import { ProgressBar } from '../components/ProgressBar'
import {
  formatDate,
  formatPhase,
  formatStatus,
  formatWordCount,
} from '../lib/format'
import type { Character, OutlineEntry } from '../types/novel'

const tabs = [
  { id: 'overview', label: 'Tổng quan' },
  { id: 'premise', label: 'Tiền đề' },
  { id: 'outline', label: 'Đề cương' },
  { id: 'characters', label: 'Nhân vật' },
] as const

type TabId = (typeof tabs)[number]['id']

export function NovelDetailPage() {
  const { slug = '' } = useParams()
  const [tab, setTab] = useState<TabId>('overview')

  const novelQuery = useQuery({
    queryKey: ['novel', slug],
    queryFn: () => fetchNovel(slug),
    enabled: !!slug,
  })

  const premiseQuery = useQuery({
    queryKey: ['premise', slug],
    queryFn: () => fetchPremise(slug),
    enabled: !!slug && tab === 'premise',
  })

  const outlineQuery = useQuery({
    queryKey: ['outline', slug],
    queryFn: () => fetchOutline(slug),
    enabled: !!slug && tab === 'outline',
  })

  const charactersQuery = useQuery({
    queryKey: ['characters', slug],
    queryFn: () => fetchCharacters(slug),
    enabled: !!slug && tab === 'characters',
  })

  if (novelQuery.isLoading) {
    return <DetailSkeleton />
  }

  if (novelQuery.isError || !novelQuery.data) {
    return (
      <div className="card border-red-800/50 text-red-300">
        Không tải được truyện: {(novelQuery.error as Error)?.message ?? 'unknown'}
      </div>
    )
  }

  const novel = novelQuery.data
  const progress = novel.progress

  return (
    <div className="max-w-4xl">
      <header className="mb-6">
        <h1 className="text-3xl font-semibold text-white">{novel.title}</h1>
        <div className="mt-2 flex flex-wrap gap-2 text-sm text-slate-400">
          <span>{formatStatus(novel.status)}</span>
          <span>·</span>
          <span>{formatPhase(novel.phase)}</span>
          {novel.run_meta?.style && (
            <>
              <span>·</span>
              <span>{novel.run_meta.style}</span>
            </>
          )}
          {novel.run_meta?.model && (
            <>
              <span>·</span>
              <span>{novel.run_meta.model}</span>
            </>
          )}
          <span>·</span>
          <span>Cập nhật {formatDate(novel.updated_at)}</span>
        </div>
        <div className="mt-4">
          <ProgressBar
            completed={novel.completed_count}
            total={novel.total_chapters}
            label="Tiến độ chương"
          />
        </div>
      </header>

      <div className="mb-6 flex gap-4 border-b border-surface-border">
        {tabs.map((t) => (
          <button
            key={t.id}
            type="button"
            onClick={() => setTab(t.id)}
            className={`pb-3 text-sm font-medium transition ${tab === t.id ? 'tab-active' : 'tab-inactive'}`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {tab === 'overview' && (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <StatCard label="Chương hiện tại" value={`${progress.current_chapter} / ${progress.total_chapters}`} />
          <StatCard label="Đã hoàn thành" value={String(progress.completed_chapters.length)} />
          <StatCard label="Tổng số từ" value={formatWordCount(progress.total_word_count)} />
          {progress.in_progress_chapter ? (
            <StatCard label="Đang viết" value={`Chương ${progress.in_progress_chapter}`} />
          ) : null}
          {progress.layered && (
            <StatCard
              label="Phân tầng"
              value={`Tập ${progress.current_volume ?? '—'} · Cung ${progress.current_arc ?? '—'}`}
            />
          )}
          {novel.run_meta?.planning_tier && (
            <StatCard label="Quy mô" value={novel.run_meta.planning_tier} />
          )}
        </div>
      )}

      {tab === 'premise' && (
        <TabPanel loading={premiseQuery.isLoading} error={premiseQuery.error}>
          {premiseQuery.data?.content ? (
            <pre className="whitespace-pre-wrap font-sans text-sm leading-relaxed text-slate-300">
              {premiseQuery.data.content}
            </pre>
          ) : (
            <EmptyState>Chưa có tiền đề.</EmptyState>
          )}
        </TabPanel>
      )}

      {tab === 'outline' && (
        <TabPanel loading={outlineQuery.isLoading} error={outlineQuery.error}>
          {outlineQuery.data?.layered && outlineQuery.data.volumes ? (
            <div className="space-y-6">
              {outlineQuery.data.volumes.map((vol) => (
                <section key={vol.index}>
                  <h3 className="mb-2 text-lg font-medium text-white">
                    Tập {vol.index}: {vol.title}
                  </h3>
                  <p className="mb-3 text-sm text-slate-400">{vol.theme}</p>
                  {vol.arcs.map((arc) => (
                    <div key={arc.index} className="mb-4 ml-2 border-l-2 border-surface-border pl-4">
                      <h4 className="font-medium text-slate-200">
                        Cung {arc.index}: {arc.title}
                      </h4>
                      <p className="mb-2 text-xs text-slate-500">{arc.goal}</p>
                      <OutlineList entries={arc.chapters} />
                    </div>
                  ))}
                </section>
              ))}
            </div>
          ) : outlineQuery.data?.entries?.length ? (
            <OutlineList entries={outlineQuery.data.entries} />
          ) : (
            <EmptyState>Chưa có đề cương.</EmptyState>
          )}
        </TabPanel>
      )}

      {tab === 'characters' && (
        <TabPanel loading={charactersQuery.isLoading} error={charactersQuery.error}>
          {charactersQuery.data?.length ? (
            <div className="space-y-4">
              {charactersQuery.data.map((c) => (
                <CharacterCard key={c.name} character={c} />
              ))}
            </div>
          ) : (
            <EmptyState>Chưa có hồ sơ nhân vật.</EmptyState>
          )}
        </TabPanel>
      )}
    </div>
  )
}

function StatCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="card">
      <div className="text-xs uppercase tracking-wide text-slate-500">{label}</div>
      <div className="mt-1 text-xl font-semibold text-white">{value}</div>
    </div>
  )
}

function TabPanel({
  loading,
  error,
  children,
}: {
  loading: boolean
  error: unknown
  children: React.ReactNode
}) {
  if (loading) return <DetailSkeleton />
  if (error) {
    return (
      <div className="card border-red-800/50 text-red-300">
        Lỗi tải dữ liệu: {(error as Error).message}
      </div>
    )
  }
  return <div className="card">{children}</div>
}

function EmptyState({ children }: { children: React.ReactNode }) {
  return <p className="text-slate-500">{children}</p>
}

function OutlineList({ entries }: { entries: OutlineEntry[] }) {
  return (
    <ul className="space-y-3">
      {entries.map((e) => (
        <li key={e.chapter} className="rounded-lg border border-surface-border/60 p-3">
          <div className="font-medium text-white">
            Ch. {e.chapter}: {e.title}
          </div>
          <p className="mt-1 text-sm text-slate-400">{e.core_event}</p>
        </li>
      ))}
    </ul>
  )
}

function CharacterCard({ character }: { character: Character }) {
  return (
    <article className="rounded-lg border border-surface-border/60 p-4">
      <div className="flex flex-wrap items-center gap-2">
        <h3 className="text-lg font-medium text-white">{character.name}</h3>
        {character.tier && (
          <span className="badge bg-accent/20 text-accent">{character.tier}</span>
        )}
        <span className="text-sm text-slate-500">{character.role}</span>
      </div>
      {character.aliases?.length ? (
        <p className="mt-1 text-xs text-slate-500">Còn gọi: {character.aliases.join(', ')}</p>
      ) : null}
      <p className="mt-2 text-sm text-slate-300">{character.description}</p>
      {character.traits?.length ? (
        <div className="mt-2 flex flex-wrap gap-1">
          {character.traits.map((t) => (
            <span key={t} className="badge bg-surface-border text-slate-400">
              {t}
            </span>
          ))}
        </div>
      ) : null}
    </article>
  )
}