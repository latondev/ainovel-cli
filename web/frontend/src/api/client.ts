import type { APIError } from '../types/novel'

export class ApiClientError extends Error {
  code: string
  status: number

  constructor(message: string, code: string, status: number) {
    super(message)
    this.name = 'ApiClientError'
    this.code = code
    this.status = status
  }
}

async function parseError(res: Response): Promise<ApiClientError> {
  try {
    const body = (await res.json()) as APIError
    return new ApiClientError(body.error || res.statusText, body.code || 'UNKNOWN', res.status)
  } catch {
    return new ApiClientError(res.statusText, 'UNKNOWN', res.status)
  }
}

export async function apiGet<T>(path: string): Promise<T> {
  const res = await fetch(path, {
    headers: { Accept: 'application/json' },
  })
  if (!res.ok) {
    throw await parseError(res)
  }
  return res.json() as Promise<T>
}