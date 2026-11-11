import { useEffect, useState } from 'react'
import { syncClock } from './clock'
import type { User } from './data'

export class APIError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
  ) {
    super(message)
  }
}
let csrf = ''
export const setCSRF = (value: string) => {
  csrf = value
}
type RequestOptions = { method?: string; body?: unknown; token?: string; signal?: AbortSignal }
export async function api<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const headers: Record<string, string> = { Accept: 'application/json' }
  if (options.body !== undefined) headers['Content-Type'] = 'application/json'
  if (csrf) headers['X-CSRF-Token'] = csrf
  if (options.token) headers.Authorization = 'Bearer ' + options.token
  let response: Response
  try {
    response = await fetch('/api/v1' + path, {
      method: options.method ?? 'GET',
      credentials: 'same-origin',
      headers,
      body: options.body === undefined ? undefined : JSON.stringify(options.body),
      signal: options.signal,
    })
  } catch (error) {
    if (error instanceof DOMException && error.name === 'AbortError') throw error
    throw new APIError(
      0,
      'NETWORK_ERROR',
      'Koneksi terputus. Periksa internet lalu coba kembali dengan isian yang sama.',
    )
  }
  syncClock(response.headers.get('X-Server-Time'))
  let data
  try {
    data = await response.json()
  } catch {
    throw new APIError(
      response.status,
      'INVALID_RESPONSE',
      'Server belum memberikan jawaban yang dapat dibaca. Coba kembali.',
    )
  }
  if (!response.ok) {
    if (response.status === 401 && path !== '/auth/login' && path !== '/auth/session')
      window.dispatchEvent(new Event('gilir:session-ended'))
    throw new APIError(
      response.status,
      data.error?.code ?? 'REQUEST_FAILED',
      data.error?.message ?? 'Permintaan belum berhasil.',
    )
  }
  return data as T
}
export async function getSession() {
  const session = await api<{ user: User; csrfToken: string }>('/auth/session')
  setCSRF(session.csrfToken)
  return session.user
}
export const messageOf = (error: unknown) =>
  error instanceof Error ? error.message : 'Permintaan belum berhasil.'

// Fetch fresh server availability when inputs change, every 30 seconds, and on focus.
export function useSlots(path: string | null, token?: string) {
  const [state, setState] = useState<{
    path: string | null
    slots: number[]
    loading: boolean
    error: string
  }>({ path: null, slots: [], loading: false, error: '' })
  const [revision, setRevision] = useState(0)
  useEffect(() => {
    const reload = () => setRevision((value) => value + 1)
    const timer = window.setInterval(reload, 30_000)
    window.addEventListener('focus', reload)
    return () => {
      window.clearInterval(timer)
      window.removeEventListener('focus', reload)
    }
  }, [])
  useEffect(() => {
    if (!path) return
    const abort = new AbortController()
    setState({ path, slots: [], loading: true, error: '' })
    api<{ slots: number[] }>(path, { token, signal: abort.signal })
      .then((result) => {
        if (!abort.signal.aborted)
          setState({ path, slots: result.slots, loading: false, error: '' })
      })
      .catch((error) => {
        if (!abort.signal.aborted)
          setState({ path, slots: [], loading: false, error: messageOf(error) })
      })
    return () => abort.abort()
  }, [path, token, revision])
  return {
    slots: state.path === path ? state.slots : [],
    loading: !!path && (state.path !== path || state.loading),
    error: state.path === path ? state.error : '',
    reload: () => setRevision((value) => value + 1),
  }
}
export function availabilityPath(
  serviceId: string,
  barberId: string,
  date: string,
  mode = 'online',
) {
  return (
    (mode === 'online' ? '/availability?' : '/admin/availability?') +
    new URLSearchParams({ serviceId, barberId, date, mode })
  )
}
