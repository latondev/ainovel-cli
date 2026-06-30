import { Link } from 'react-router-dom'
import { BookOpen } from 'lucide-react'
import type { NovelSummary } from '../types/novel'
import { formatPhase, formatStatus, formatWordCount } from '../lib/format'
import { ProgressBar } from './ProgressBar'

const statusColors: Record<string, string> = {
  idle: 'bg-slate-600/40 text-slate-300',
  running: 'bg-emerald-600/30 text-emerald-300',
  done: 'bg-blue-600/30 text-blue-300',
  error: 'bg-red-600/30 text-red-300',
  stopped: 'bg-amber-600/30 text-amber-300',
  queued: 'bg-purple-600/30 text-purple-300',
}

export function NovelCard({ novel }: { novel: NovelSummary }) {
  return (
    <Link to={`/novels/${novel.slug}`} className="card block transition hover:border-accent/50">
      <div className="flex items-start gap-4">
        <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-lg bg-accent/20 text-accent">
          <BookOpen size={22} />
        </div>
        <div className="min-w-0 flex-1 space-y-2">
          <div className="flex flex-wrap items-center gap-2">
            <h2 className="truncate text-lg font-semibold text-white">{novel.title}</h2>
            <span className={`badge ${statusColors[novel.status] ?? statusColors.idle}`}>
              {formatStatus(novel.status)}
            </span>
            <span className="badge bg-surface-border text-slate-400">{formatPhase(novel.phase)}</span>
          </div>
          <p className="text-sm text-slate-400">
            Chương {novel.current_chapter}/{novel.total_chapters} · {formatWordCount(novel.word_count)} từ
            {novel.model ? ` · ${novel.model}` : ''}
          </p>
          <ProgressBar
            completed={novel.completed_count}
            total={novel.total_chapters}
          />
        </div>
      </div>
    </Link>
  )
}