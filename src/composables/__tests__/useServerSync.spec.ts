import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { installFakeServer, type FakeServer } from '@/__tests__/fakeServer'
import { createSeedState } from '@/seed'
import { useKanbanStore } from '@/stores/kanban'

let server: FakeServer

/** A store that has loaded the demo state from the fake server. */
async function loadedStore() {
  const store = useKanbanStore()
  await store.ensureLoaded()
  server.requests.length = 0 // forget the load; tests look at what the actions send
  return store
}

function firstColumn(store: ReturnType<typeof useKanbanStore>) {
  const board = store.boardList[0]
  const column = board && store.columnsForBoard(board.id)[0]
  if (!board || !column) throw new Error('seed is empty')
  return { board, column }
}

describe('server sync (through the kanban store)', () => {
  beforeEach(() => {
    server = installFakeServer()
    setActivePinia(createPinia())
  })

  afterEach(() => {
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
      expect(store.version).toBe(1)
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
      server.state = { boards: {}, columns: {} } as unknown as typeof server.state
      const store = useKanbanStore()

      await store.ensureLoaded()

      expect(store.status).toBe('error')
    })
  })

  describe('actions update the screen first, then tell the server', () => {
    it('adds a board locally before the server has answered, with the server using its id', async () => {
      const store = await loadedStore()

      const board = store.addBoard('Roadmap')

      expect(store.state.boards[board.id]?.title).toBe('Roadmap') // already on screen
      expect(server.requests).toEqual([]) // and the request is only queued
      await store.settled()
      expect(server.requests).toEqual(['POST /api/boards'])
      expect(server.state.boards[board.id]?.title).toBe('Roadmap')
      expect(store.version).toBe(server.version)
    })

    it('sends one request per action, in the order they were made', async () => {
      const store = await loadedStore()
      const { board, column } = firstColumn(store)

      const card = store.addCard(column.id, 'New card')
      store.updateCard(card?.id ?? '', { title: 'Renamed', labels: ['ui'], dueDate: null })
      store.renameColumn(column.id, 'Backlog')
      store.renameBoard(board.id, 'Main')
      store.moveCard(card?.id ?? '', column.id, 0)
      store.deleteCard(card?.id ?? '')
      await store.settled()

      expect(server.requests).toEqual([
        `POST /api/columns/${column.id}/cards`,
        `PATCH /api/cards/${card?.id}`,
        `PATCH /api/columns/${column.id}`,
        `PATCH /api/boards/${board.id}`,
        `PUT /api/cards/${card?.id}/position`,
        `DELETE /api/cards/${card?.id}`,
      ])
      expect(server.state.columns[column.id]?.title).toBe('Backlog')
      expect(server.state.boards[board.id]?.title).toBe('Main')
      expect(server.state.cards[card?.id ?? '']).toBeUndefined()
    })

    it('sends a board, its columns and its cards being deleted as one request', async () => {
      const store = await loadedStore()
      const { board } = firstColumn(store)

      store.deleteBoard(board.id)
      await store.settled()

      expect(server.requests).toEqual([`DELETE /api/boards/${board.id}`])
      expect(store.boardList).toEqual([])
      expect(server.state.cards).toEqual({})
    })

    it('adds a column and moves a card between columns', async () => {
      const store = await loadedStore()
      const { board, column } = firstColumn(store)
      const column2 = store.addColumn(board.id, 'Extra')
      const cardId = column.cardIds[0] ?? ''

      store.moveCard(cardId, column2?.id ?? '', 0)
      await store.settled()

      expect(server.state.columns[column2?.id ?? '']?.cardIds).toEqual([cardId])
      expect(server.state.columns[column.id]?.cardIds).not.toContain(cardId)
      expect(server.state.boards[board.id]?.columnIds).toContain(column2?.id)
    })

    it('does not send anything for an action on something that does not exist', async () => {
      const store = await loadedStore()

      store.renameBoard('nope', 'x')
      store.deleteCard('nope')
      store.moveCard('nope', 'nope', 0)
      store.addCard('nope', 'x')
      await store.settled()

      expect(server.requests).toEqual([])
    })

    it('keeps its version in step with the server', async () => {
      const store = await loadedStore()

      store.addBoard('One')
      store.addBoard('Two')
      await store.settled()

      expect(store.version).toBe(3)
      expect(server.version).toBe(3)
    })

    it('replaces the whole state for an import and for a reset', async () => {
      const store = await loadedStore()
      const demo = createSeedState()

      expect(store.importState(demo)).toBe(true)
      await store.settled()
      expect(server.requests).toEqual(['PUT /api/state'])
      expect(Object.keys(server.state.boards)).toEqual(Object.keys(demo.boards))

      store.resetToDemo()
      await store.settled()
      expect(server.requests).toEqual(['PUT /api/state', 'PUT /api/state'])
      expect(store.version).toBe(server.version)
    })

    it('does not send an import that fails validation', async () => {
      const store = await loadedStore()

      expect(store.importState({ boards: {}, columns: {} })).toBe(false)
      await store.settled()

      expect(server.requests).toEqual([])
    })

    it('reloads when the server used a different id than the one proposed', async () => {
      const store = await loadedStore()
      server.ignoreClientIds = true

      const board = store.addBoard('Mine')
      await store.settled()
      // The reload runs on its own once the queue is empty.
      await vi.waitFor(() => expect(store.state.boards[board.id]).toBeUndefined())

      expect(store.boardList.map((item) => item.title)).toContain('Mine')
      expect(store.notice).toBeNull()
    })

    it('survives a refresh: a new store loads what the actions saved', async () => {
      const first = await loadedStore()
      first.addBoard('Persisted')
      await first.settled()

      setActivePinia(createPinia())
      const reloaded = useKanbanStore()
      await reloaded.ensureLoaded()

      expect(reloaded.boardList.map((board) => board.title)).toEqual(['Vue Kanban', 'Persisted'])
    })
  })

  describe('when a request fails', () => {
    it('reloads the server state, tells the user, and skips what was queued behind it', async () => {
      const store = await loadedStore()
      const { board } = firstColumn(store)
      server.failNextWriteWith = 500

      store.renameBoard(board.id, 'Never saved')
      store.addBoard('Queued behind the failure')
      await store.settled()
      await vi.waitFor(() => expect(store.notice).not.toBeNull())

      expect(store.notice).toMatch(/couldn't save/i)
      expect(store.boardList.map((item) => item.title)).toEqual(['Vue Kanban']) // rolled back
      expect(server.requests.filter((request) => request.startsWith('POST'))).toEqual([])
      expect(server.state.boards[board.id]?.title).toBe('Vue Kanban')
    })

    it('works again after recovering', async () => {
      const store = await loadedStore()
      server.failNextWriteWith = 500
      store.addBoard('Lost')
      await store.settled()

      store.addBoard('Kept')
      await store.settled()

      expect(Object.values(server.state.boards).map((board) => board.title)).toEqual([
        'Vue Kanban',
        'Kept',
      ])
    })

    it('explains a 404 as the item having been deleted elsewhere', async () => {
      const store = await loadedStore()
      const { column } = firstColumn(store)
      const cardId = column.cardIds[0] ?? ''
      delete server.state.cards[cardId] // another tab deleted it
      server.state.columns[column.id]?.cardIds.splice(0, 1)

      store.updateCard(cardId, { title: 'Too late' })
      await store.settled()

      expect(store.notice).toMatch(/no longer exists|deleted/i)
    })

    it('keeps the local state and says so when the server cannot be reached', async () => {
      const store = await loadedStore()
      server.online = false

      store.addBoard('Offline')
      await store.settled()

      expect(store.notice).toMatch(/reach the server/i)
      expect(store.boardList.map((board) => board.title)).toContain('Offline')
    })

    it('reports a rejected whole-state replace as a conflict', async () => {
      const store = await loadedStore()
      server.version += 1 // another tab saved first

      store.resetToDemo()
      await store.settled()

      expect(store.notice).toMatch(/conflict/i)
      expect(store.version).toBe(server.version)
    })

    it('can be dismissed', async () => {
      const store = await loadedStore()
      server.failNextWriteWith = 500
      store.addBoard('x')
      await store.settled()

      store.dismissNotice()

      expect(store.notice).toBeNull()
    })
  })

  describe('changes made elsewhere', () => {
    it('notices a gap in the versions and reloads, without losing a change still on its way', async () => {
      const store = await loadedStore()
      // Another tab adds a board straight on the server.
      const elsewhereId = crypto.randomUUID()
      server.state.boards[elsewhereId] = { id: elsewhereId, title: 'Elsewhere', columnIds: [] }
      server.version += 1

      store.addBoard('Mine') // reply says version 3 while we were at 1: a gap
      await store.settled()
      await vi.waitFor(() =>
        expect(store.boardList.map((board) => board.title)).toContain('Elsewhere'),
      )

      expect(store.boardList.map((board) => board.title)).toEqual([
        'Vue Kanban',
        'Elsewhere',
        'Mine',
      ])
      expect(store.version).toBe(server.version)
    })

    it('refreshIfBehind loads only when the server is ahead', async () => {
      const store = await loadedStore()

      await store.refreshIfBehind(store.version)
      expect(server.requests).toEqual([])

      server.version += 1
      await store.refreshIfBehind(server.version)
      expect(server.requests).toEqual(['GET /api/state'])
      expect(store.version).toBe(server.version)
    })

    it('never replaces the state while a local change is still waiting to be sent', async () => {
      const store = await loadedStore()

      store.addBoard('In flight') // queued, not yet on the server
      await store.load() // a reload has to wait for it, or it would wipe it out

      expect(store.boardList.map((board) => board.title)).toContain('In flight')
      expect(Object.values(server.state.boards).map((board) => board.title)).toContain('In flight')
    })
  })
})
