import { formatPercent } from '../lib/format'

type Props = {
  completed: number
  total: number
  label?: string
}

export function ProgressBar({ completed, total, label }: Props) {
  const pct = formatPercent(completed, total)
  return (
    <div className="space-y-1">
      {label && (
        <div className="flex justify-between text-xs text-slate-400">
          <span>{label}</span>
          <span>
            {completed}/{total} ({pct}%)
          </span>
        </div>
      )}
      <div className="h-2 overflow-hidden rounded-full bg-surface-border">
        <div
          className="h-full rounded-full bg-accent transition-all duration-500"
          style={{ width: `${pct}%` }}
        />
      </div>
    </div>
  )
}