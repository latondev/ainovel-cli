export type NovelSummary = {
  slug: string
  title: string
  status: string
  phase: string
  current_chapter: number
  total_chapters: number
  completed_count: number
  word_count: number
  style?: string
  model?: string
  updated_at: number
}

export type Progress = {
  novel_name: string
  phase: string
  current_chapter: number
  total_chapters: number
  completed_chapters: number[]
  total_word_count: number
  chapter_word_counts?: Record<string, number>
  in_progress_chapter?: number
  flow?: string
  layered?: boolean
  current_volume?: number
  current_arc?: number
}

export type RunMeta = {
  started_at?: string
  provider?: string
  style?: string
  model?: string
  planning_tier?: string
}

export type NovelDetail = NovelSummary & {
  progress: Progress
  run_meta?: RunMeta
}

export type PremiseResponse = {
  content: string
}

export type OutlineEntry = {
  chapter: number
  title: string
  core_event: string
  hook: string
  scenes?: string[]
}

export type ArcOutline = {
  index: number
  title: string
  goal: string
  estimated_chapters?: number
  chapters: OutlineEntry[]
}

export type VolumeOutline = {
  index: number
  title: string
  theme: string
  arcs: ArcOutline[]
}

export type OutlineResponse = {
  layered: boolean
  entries?: OutlineEntry[]
  volumes?: VolumeOutline[]
}

export type Character = {
  name: string
  aliases?: string[]
  role: string
  description: string
  arc: string
  traits?: string[]
  tier?: string
}

export type APIError = {
  error: string
  code: string
}