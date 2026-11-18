import type { CardPatch, ID, KanbanState } from './types'
import { isKanbanState } from './validation'

/** The whole state with the version it was read at (see docs/api.md). */
export interface Snapshot {
  version: number
  state: KanbanState
}

/** What every successful write answers with: the version after the change. */
export interface Written {
  version: number
  /** For creating requests: the id the server used, which can differ from the one proposed. */
  id?: ID
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

const unexpected = () => new ApiError(0, 'The server sent data the app does not understand.')

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
  if (!isSnapshot(body)) throw unexpected()
  return body
}

function withBody(method: string, body?: object): RequestInit {
  return {
    method,
    headers: { 'Content-Type': 'application/json' },
    body: body === undefined ? undefined : JSON.stringify(body),
  }
}

function versionOf(body: unknown): number {
  if (typeof body === 'object' && body !== null && 'version' in body) {
    if (typeof body.version === 'number') return body.version
  }
  throw unexpected()
}

/** The id of the entity under `key` in a creating request's reply, e.g. `{ board: { id } }`. */
function createdId(body: unknown, key: string): ID {
  if (typeof body === 'object' && body !== null && key in body) {
    const entity = (body as Record<string, unknown>)[key]
    if (typeof entity === 'object' && entity !== null && 'id' in entity) {
      if (typeof entity.id === 'string') return entity.id
    }
  }
  throw unexpected()
}

async function write(path: string, init: RequestInit): Promise<Written> {
  return { version: versionOf(await request(path, init)) }
}

async function create(path: string, key: string, id: ID, title: string): Promise<Written> {
  const body = await request(path, withBody('POST', { id, title }))
  return { version: versionOf(body), id: createdId(body, key) }
}

/** Replaces the whole state. Rejects with a 409 ApiError if `version` is no longer current. */
export function putState(version: number, state: KanbanState): Promise<Written> {
  return write('/api/state', withBody('PUT', { version, state }))
}

// The per-action endpoints. Each one mirrors a store action (see docs/api.md).

export const addBoard = (id: ID, title: string) => create('/api/boards', 'board', id, title)
export const renameBoard = (id: ID, title: string) =>
  write(`/api/boards/${id}`, withBody('PATCH', { title }))
export const deleteBoard = (id: ID) => write(`/api/boards/${id}`, withBody('DELETE'))

export const addColumn = (boardId: ID, id: ID, title: string) =>
  create(`/api/boards/${boardId}/columns`, 'column', id, title)
export const renameColumn = (id: ID, title: string) =>
  write(`/api/columns/${id}`, withBody('PATCH', { title }))
export const deleteColumn = (id: ID) => write(`/api/columns/${id}`, withBody('DELETE'))

export const addCard = (columnId: ID, id: ID, title: string) =>
  create(`/api/columns/${columnId}/cards`, 'card', id, title)
export const updateCard = (id: ID, patch: CardPatch) =>
  write(`/api/cards/${id}`, withBody('PATCH', patch))
export const deleteCard = (id: ID) => write(`/api/cards/${id}`, withBody('DELETE'))
export const moveCard = (id: ID, columnId: ID, index: number) =>
  write(`/api/cards/${id}/position`, withBody('PUT', { columnId, index }))
