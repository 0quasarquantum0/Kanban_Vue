import { vi } from 'vitest'
import { createSeedState } from '@/seed'
import type { Card, Column, KanbanState } from '@/types'
import { isKanbanState } from '@/validation'

/** What the fake Go server knows, and what tests can poke at. */
export interface FakeServer {
  version: number
  state: KanbanState
  /** When false, every request fails like a refused connection. */
  online: boolean
  /** One "METHOD /path" entry per request received, in order. */
  requests: string[]
  /** The next write request is answered with this status, and changes nothing. */
  failNextWriteWith: number | null
  /** When true, ids proposed by the client are ignored and new ones are made. */
  ignoreClientIds: boolean
}

function json(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

type Body = Record<string, unknown>

/**
 * Replaces the global `fetch` with a stand-in for the Go server (docs/api.md): the whole-state
 * endpoints with their version check, and the per-action endpoints. Tests read and change the
 * returned object to set up a scenario, or to play another browser tab.
 */
export function installFakeServer(state: KanbanState = createSeedState()): FakeServer {
  const server: FakeServer = {
    version: 1,
    state,
    online: true,
    requests: [],
    failNextWriteWith: null,
    ignoreClientIds: false,
  }

  const missing = (kind: string, id: string) => json({ error: `${kind} "${id}": not found` }, 404)
  const idFor = (body: Body) =>
    typeof body.id === 'string' && !server.ignoreClientIds ? body.id : crypto.randomUUID()
  const wrote = (extra: Body = {}, status = 200) => {
    server.version += 1
    return json({ version: server.version, ...extra }, status)
  }

  function write(method: string, path: string, body: Body): Response {
    const s = server.state
    let m: RegExpMatchArray | null

    if (method === 'PUT' && path === '/api/state') {
      if (body.version !== server.version) return json({ error: 'version conflict' }, 409)
      if (!isKanbanState(body.state)) return json({ error: 'invalid state' }, 400)
      server.state = body.state
      return wrote()
    }
    if (method === 'POST' && path === '/api/boards') {
      const board = { id: idFor(body), title: String(body.title), columnIds: [] }
      s.boards[board.id] = board
      return wrote({ board }, 201)
    }
    if ((m = path.match(/^\/api\/boards\/([^/]+)$/))) {
      const board = s.boards[m[1] ?? '']
      if (!board) return missing('board', m[1] ?? '')
      if (method === 'PATCH') board.title = String(body.title)
      else {
        for (const columnId of board.columnIds) deleteColumn(columnId)
        delete s.boards[board.id]
      }
      return wrote()
    }
    if ((m = path.match(/^\/api\/boards\/([^/]+)\/columns$/)) && method === 'POST') {
      const board = s.boards[m[1] ?? '']
      if (!board) return missing('board', m[1] ?? '')
      const column: Column = { id: idFor(body), title: String(body.title), cardIds: [] }
      s.columns[column.id] = column
      board.columnIds.push(column.id)
      return wrote({ column }, 201)
    }
    if ((m = path.match(/^\/api\/columns\/([^/]+)$/))) {
      const column = s.columns[m[1] ?? '']
      if (!column) return missing('column', m[1] ?? '')
      if (method === 'PATCH') column.title = String(body.title)
      else deleteColumn(column.id)
      return wrote()
    }
    if ((m = path.match(/^\/api\/columns\/([^/]+)\/cards$/)) && method === 'POST') {
      const column = s.columns[m[1] ?? '']
      if (!column) return missing('column', m[1] ?? '')
      const card: Card = {
        id: idFor(body),
        title: String(body.title),
        description: '',
        labels: [],
        dueDate: null,
        createdAt: new Date().toISOString(),
      }
      s.cards[card.id] = card
      column.cardIds.push(card.id)
      return wrote({ card }, 201)
    }
    if ((m = path.match(/^\/api\/cards\/([^/]+)$/))) {
      const card = s.cards[m[1] ?? '']
      if (!card) return missing('card', m[1] ?? '')
      if (method === 'PATCH') Object.assign(card, body)
      else {
        detach(card.id)
        delete s.cards[card.id]
      }
      return wrote({ card })
    }
    if ((m = path.match(/^\/api\/cards\/([^/]+)\/position$/)) && method === 'PUT') {
      const card = s.cards[m[1] ?? '']
      if (!card) return missing('card', m[1] ?? '')
      const to = s.columns[String(body.columnId)]
      if (!to) return missing('column', String(body.columnId))
      detach(card.id)
      to.cardIds.splice(Math.max(0, Math.min(Number(body.index), to.cardIds.length)), 0, card.id)
      return wrote()
    }
    return json({ error: 'not found' }, 404)
  }

  function detach(cardId: string) {
    for (const column of Object.values(server.state.columns)) {
      const index = column.cardIds.indexOf(cardId)
      if (index !== -1) column.cardIds.splice(index, 1)
    }
  }

  function deleteColumn(columnId: string) {
    const column = server.state.columns[columnId]
    if (!column) return
    for (const cardId of column.cardIds) delete server.state.cards[cardId]
    for (const board of Object.values(server.state.boards)) {
      board.columnIds = board.columnIds.filter((id) => id !== columnId)
    }
    delete server.state.columns[columnId]
  }

  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const method = init?.method ?? 'GET'
      const path = String(input)
      server.requests.push(`${method} ${path}`)
      if (!server.online) throw new TypeError('Failed to fetch')

      if (method === 'GET') {
        if (path === '/api/state') return json({ version: server.version, state: server.state })
        return json({ error: 'not found' }, 404)
      }
      if (server.failNextWriteWith !== null) {
        const status = server.failNextWriteWith
        server.failNextWriteWith = null
        return json({ error: 'forced failure' }, status)
      }
      const body = (init?.body ? JSON.parse(String(init.body)) : {}) as Body
      return write(method, path, body)
    }),
  )
  return server
}
