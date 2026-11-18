import type { KanbanState } from './types'
import { isKanbanState } from './validation'

/** The whole state with the version it was read at (see docs/api.md). */
export interface Snapshot {
  version: number
  state: KanbanState
}

/** A failed request. `status` is the HTTP status, or 0 when the server could not be reached. */
export class ApiError extends Error {
  readonly status: number

  constructor(status: number, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

function errorMessage(body: unknown): string | undefined {
  if (typeof body === 'object' && body !== null && 'error' in body) {
    const { error } = body
    if (typeof error === 'string') return error
  }
  return undefined
}

/** Sends a request and returns the parsed JSON body, or throws an ApiError. */
async function request(path: string, init?: RequestInit): Promise<unknown> {
  let response: Response
  try {
    response = await fetch(path, init)
  } catch {
    throw new ApiError(0, 'Could not reach the server.')
  }

  // A proxy error page or an empty body is not JSON; the status code still says what happened.
  const body: unknown = await response.json().catch(() => null)
  if (!response.ok) {
    const fallback =
      response.status >= 500
        ? `The server is not responding properly (HTTP ${response.status}).`
        : `The request failed (HTTP ${response.status}).`
    throw new ApiError(response.status, errorMessage(body) ?? fallback)
  }
  return body
}

function isSnapshot(value: unknown): value is Snapshot {
  return (
    typeof value === 'object' &&
    value !== null &&
    'version' in value &&
    typeof value.version === 'number' &&
    'state' in value &&
    isKanbanState(value.state)
  )
}

export async function getState(): Promise<Snapshot> {
  const body = await request('/api/state')
  if (!isSnapshot(body)) throw new ApiError(0, 'The server sent data the app does not understand.')
  return body
}

/** Replaces the whole state. Rejects with a 409 ApiError if `version` is no longer current. */
export async function putState(version: number, state: KanbanState): Promise<number> {
  const body = await request('/api/state', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ version, state }),
  })
  if (typeof body !== 'object' || body === null || !('version' in body)) {
    throw new ApiError(0, 'The server sent data the app does not understand.')
  }
  const next = body.version
  if (typeof next !== 'number') {
    throw new ApiError(0, 'The server sent data the app does not understand.')
  }
  return next
}
