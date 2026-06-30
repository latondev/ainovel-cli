const phaseLabels: Record<string, string> = {
  init: 'Khởi tạo',
  premise: 'Tiền đề',
  outline: 'Đề cương',
  writing: 'Đang viết',
  complete: 'Hoàn thành',
}

const statusLabels: Record<string, string> = {
  idle: 'Tạm dừng',
  running: 'Đang chạy',
  queued: 'Chờ',
  done: 'Xong',
  error: 'Lỗi',
  stopped: 'Đã dừng',
}

export function formatPhase(phase: string): string {
  return phaseLabels[phase] ?? phase
}

export function formatStatus(status: string): string {
  return statusLabels[status] ?? status
}

export function formatWordCount(n: number): string {
  return new Intl.NumberFormat('vi-VN').format(n)
}

export function formatPercent(completed: number, total: number): number {
  if (total <= 0) return 0
  return Math.round((completed / total) * 100)
}

export function formatDate(ts: number): string {
  if (!ts) return '—'
  return new Intl.DateTimeFormat('vi-VN', {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(ts * 1000))
}