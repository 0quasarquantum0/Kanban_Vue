import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { installFakeServer, type FakeServer } from '@/__tests__/fakeServer'
import { useKanbanStore } from '@/stores/kanban'
import type { KanbanState } from '@/types'

let server: FakeServer

function boardTitles(state: KanbanState): string[] {
  return Object.values(state.boards).map((board) => board.title)
}

describe('server sync (through the kanban store)', () => {
  beforeEach(() => {
    server = installFakeServer()
    setActivePinia(createPinia())
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  describe('loading', () => {
    it('starts idle with an empty board list', () => {
      const store = useKanbanStore()

      expect(store.status).toBe('idle')
      expect(store.boardList).toEqual([])
    })

    it('loads the state from the server and becomes ready', async () => {
      const store = useKanbanStore()

      const loading = store.ensureLoaded()
      expect(store.status).toBe('loading')
      await loading

      expect(store.status).toBe('ready')
      expect(store.boardList.map((board) => board.title)).toEqual(['Vue Kanban'])
      expect(server.requests).toEqual(['GET /api/state'])
    })

    it('only fetches once however many callers wait for the first load', async () => {
      const store = useKanbanStore()

      await Promise.all([store.ensureLoaded(), store.ensureLoaded(), store.ensureLoaded()])

      expect(server.requests).toEqual(['GET /api/state'])
    })

    it('reports an error when the server cannot be reached', async () => {
      server.online = false
      const store = useKanbanStore()

      await store.ensureLoaded()

      expect(store.status).toBe('error')
      expect(store.loadError).toMatch(/reach the server/i)
    })

    it('can load again after an error', async () => {
      server.online = false
      const store = useKanbanStore()
      await store.ensureLoaded()

      server.online = true
      await store.load()

      expect(store.status).toBe('ready')
      expect(store.loadError).toBeNull()
    })

    it('rejects a response that is not a valid state', async () => {
      server.state = { boards: {}, columns: {} } as unknown as KanbanState
      const store = useKanbanStore()

      await store.ensureLoaded()

      expect(store.status).toBe('error')
    })
  })

  describe('saving', () => {
    it('sends nothing when nothing changed', async () => {
      const store = useKanbanStore()
      await store.ensureLoaded()

      await store.saveNow()

      expect(server.requests).toEqual(['GET /api/state'])
    })

    it('does not save the state it has just loaded', async () => {
      vi.useFakeTimers()
      const store = useKanbanStore()
      await store.ensureLoaded()

      await vi.advanceTimersByTimeAsync(1000)

      expect(server.requests).toEqual(['GET /api/state'])
    })

    it('saves a change to the server and moves to the new version', async () => {
      const store = useKanbanStore()
      await store.ensureLoaded()

      store.addBoard('Saved')
      await store.saveNow()

      expect(boardTitles(server.state)).toEqual(['Vue Kanban', 'Saved'])
      expect(server.version).toBe(2)
      expect(store.version).toBe(2)
    })

    it('saves by itself shortly after a change (debounced)', async () => {
      vi.useFakeTimers()
      const store = useKanbanStore()
      await store.ensureLoaded()

      store.addBoard('One')
      store.addBoard('Two')
      await vi.advanceTimersByTimeAsync(100)
      expect(server.requests.filter((request) => request.startsWith('PUT'))).toHaveLength(0)
      await vi.advanceTimersByTimeAsync(1000)

      expect(server.requests.filter((request) => request.startsWith('PUT'))).toHaveLength(1)
      expect(boardTitles(server.state)).toEqual(['Vue Kanban', 'One', 'Two'])
    })

    it('keeps the saved changes after a reload (a new store stands in for a refresh)', async () => {
      const first = useKanbanStore()
      await first.ensureLoaded()
      first.addBoard('Persisted')
      await first.saveNow()

      setActivePinia(createPinia())
      const reloaded = useKanbanStore()
      await reloaded.ensureLoaded()

      expect(reloaded.boardList.map((board) => board.title)).toEqual(['Vue Kanban', 'Persisted'])
    })

    it('tells the user when saving fails and tries again on the next change', async () => {
      const store = useKanbanStore()
      await store.ensureLoaded()

      server.online = false
      store.addBoard('Offline')
      await store.saveNow()
      expect(store.notice).toMatch(/couldn't save/i)

      server.online = true
      store.addBoard('Back online')
      await store.saveNow()

      expect(store.notice).toBeNull()
      expect(boardTitles(server.state)).toEqual(['Vue Kanban', 'Offline', 'Back online'])
    })
  })

  describe('conflicts between tabs', () => {
    it('reloads the server state and says so when the version is stale', async () => {
      const store = useKanbanStore()
      await store.ensureLoaded()

      // Another tab saved first.
      server.version += 1
      server.state = { ...server.state, boards: {}, columns: {}, cards: {} }

      store.addBoard('Mine')
      await store.saveNow()

      expect(store.notice).toMatch(/another tab|elsewhere|conflict/i)
      expect(store.boardList).toEqual([])
      expect(store.version).toBe(server.version)
      expect(boardTitles(server.state)).toEqual([])
    })

    it('can be dismissed', async () => {
      const store = useKanbanStore()
      await store.ensureLoaded()
      server.version += 1
      store.addBoard('Mine')
      await store.saveNow()

      store.dismissNotice()

      expect(store.notice).toBeNull()
    })
  })
})
