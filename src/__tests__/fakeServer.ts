import { vi } from 'vitest'
import { createSeedState } from '@/seed'
import type { KanbanState } from '@/types'
import { isKanbanState } from '@/validation'

/** What the fake Go server knows, and what tests can poke at. */
export interface FakeServer {
  version: number
  state: KanbanState
  /** When false, every request fails like a refused connection. */
  online: boolean
  /** One "METHOD /path" entry per request received. */
  requests: string[]
}

function json(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

/**
 * Replaces the global `fetch` with a stand-in for the Go server's whole-state endpoints
 * (docs/api.md): GET and PUT /api/state, with the version check. Tests read and change the
 * returned object to set up a scenario, or to play another browser tab.
 */
export function installFakeServer(state: KanbanState = createSeedState()): FakeServer {
  const server: FakeServer = { version: 1, state, online: true, requests: [] }

  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const method = init?.method ?? 'GET'
      server.requests.push(`${method} ${String(input)}`)
      if (!server.online) throw new TypeError('Failed to fetch')

      if (String(input) === '/api/state' && method === 'GET') {
        return json({ version: server.version, state: server.state })
      }
      if (String(input) === '/api/state' && method === 'PUT') {
        const body: unknown = JSON.parse(String(init?.body))
        const { version, state: next } = body as { version: number; state: unknown }
        if (version !== server.version) return json({ error: 'version conflict' }, 409)
        if (!isKanbanState(next)) return json({ error: 'invalid state' }, 400)
        server.state = next
        server.version += 1
        return json({ version: server.version })
      }
      return json({ error: 'not found' }, 404)
    }),
  )
  return server
}
