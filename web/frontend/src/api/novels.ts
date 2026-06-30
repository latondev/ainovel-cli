import { apiGet } from './client'
import type {
  Character,
  NovelDetail,
  NovelSummary,
  OutlineResponse,
  PremiseResponse,
} from '../types/novel'

export function fetchNovels() {
  return apiGet<NovelSummary[]>('/api/novels')
}

export function fetchNovel(slug: string) {
  return apiGet<NovelDetail>(`/api/novels/${slug}`)
}

export function fetchPremise(slug: string) {
  return apiGet<PremiseResponse>(`/api/novels/${slug}/premise`)
}

export function fetchOutline(slug: string) {
  return apiGet<OutlineResponse>(`/api/novels/${slug}/outline`)
}

export function fetchCharacters(slug: string) {
  return apiGet<Character[]>(`/api/novels/${slug}/characters`)
}