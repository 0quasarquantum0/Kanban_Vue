import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { createSeedState } from '@/seed'
import { installFakeServer } from '@/__tests__/fakeServer'
import { useKanbanStore } from '../kanban'

/** A store that has loaded the server's demo state, with a new board holding two columns. */
async function setup() {
  const store = useKanbanStore()
  await store.ensureLoaded()
  const board = store.addBoard('Test board')

  const a = store.addColumn(board.id, 'A')
  const b = store.addColumn(board.id, 'B')
  if (!a || !b) throw new Error('addColumn failed')

  const add = (columnId: string, title: string) => {
    const card = store.addCard(columnId, title)
    if (!card) throw new Error('addCard failed')
    return card.id
  }
  const titles = (columnId: string) => store.cardsForColumn(columnId).map((card) => card.title)

  return { store, board, a, b, add, titles }
}

describe('kanban store', () => {
  beforeEach(() => {
    installFakeServer()
    setActivePinia(createPinia())
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  describe('addCard', () => {
    it('appends a card with defaults to the column', async () => {
      const { store, a, add, titles } = await setup()
      add(a.id, 'first')
      const id = add(a.id, 'second')

      expect(titles(a.id)).toEqual(['first', 'second'])
      expect(store.state.cards[id]).toMatchObject({
        id,
        title: 'second',
        description: '',
        labels: [],
        dueDate: null,
      })
    })

    it('does nothing for an unknown column', async () => {
      const { store } = await setup()
      const before = Object.keys(store.state.cards).length

      expect(store.addCard('nope', 'x')).toBeUndefined()
      expect(Object.keys(store.state.cards)).toHaveLength(before)
    })
  })

  describe('moveCard', () => {
    it('reorders within a column', async () => {
      const { store, a, add, titles } = await setup()
      const one = add(a.id, '1')
      add(a.id, '2')
      add(a.id, '3')

      store.moveCard(one, a.id, 2)
      expect(titles(a.id)).toEqual(['2', '3', '1'])

      store.moveCard(one, a.id, 0)
      expect(titles(a.id)).toEqual(['1', '2', '3'])
    })

    it('moves between columns at the given index', async () => {
      const { store, a, b, add, titles } = await setup()
      const one = add(a.id, '1')
      add(a.id, '2')
      add(b.id, 'x')
      add(b.id, 'y')

      store.moveCard(one, b.id, 1)

      expect(titles(a.id)).toEqual(['2'])
      expect(titles(b.id)).toEqual(['x', '1', 'y'])
    })

    it('clamps an out-of-range index', async () => {
      const { store, a, b, add, titles } = await setup()
      const one = add(a.id, '1')
      add(b.id, 'x')

      store.moveCard(one, b.id, 99)
      expect(titles(b.id)).toEqual(['x', '1'])

      store.moveCard(one, b.id, -5)
      expect(titles(b.id)).toEqual(['1', 'x'])
    })

    it('leaves the card in place for an unknown target column', async () => {
      const { store, a, add, titles } = await setup()
      const one = add(a.id, '1')

      store.moveCard(one, 'nope', 0)
      expect(titles(a.id)).toEqual(['1'])
    })
  })

  describe('deleteColumn', () => {
    it('removes the column, its cards, and its place on the board', async () => {
      const { store, board, a, b, add } = await setup()
      const one = add(a.id, '1')
      const two = add(a.id, '2')
      const kept = add(b.id, 'kept')

      store.deleteColumn(a.id)

      expect(store.state.columns[a.id]).toBeUndefined()
      expect(board.columnIds).not.toContain(a.id)
      expect(board.columnIds).toContain(b.id)
      expect(store.state.cards[one]).toBeUndefined()
      expect(store.state.cards[two]).toBeUndefined()
      expect(store.state.cards[kept]).toBeDefined()
    })
  })

  describe('labelsForBoard', () => {
    it('lists each label on the board once, sorted', async () => {
      const { store, board, a, b, add } = await setup()
      store.updateCard(add(a.id, '1'), { labels: ['ui', 'bug'] })
      store.updateCard(add(b.id, '2'), { labels: ['bug', 'api'] })

      expect(store.labelsForBoard(board.id)).toEqual(['api', 'bug', 'ui'])
    })
  })

  describe('deleteBoard', () => {
    it('removes the board with its columns and cards, and leaves other boards alone', async () => {
      const { store, board, a, b, add } = await setup()
      const card = add(a.id, '1')
      const seedBoardIds = store.boardList
        .filter((item) => item.id !== board.id)
        .map((item) => item.id)
      const seedCardCount = Object.keys(store.state.cards).length - 1

      store.deleteBoard(board.id)

      expect(store.boardList.map((item) => item.id)).toEqual(seedBoardIds)
      expect(store.state.columns[a.id]).toBeUndefined()
      expect(store.state.columns[b.id]).toBeUndefined()
      expect(store.state.cards[card]).toBeUndefined()
      expect(Object.keys(store.state.cards)).toHaveLength(seedCardCount)
    })
  })

  describe('importState', () => {
    it('replaces the state with valid data', async () => {
      const store = useKanbanStore()
      await store.ensureLoaded()
      const next = createSeedState()
      const boardId = Object.keys(next.boards)[0]

      expect(store.importState(next)).toBe(true)
      expect(store.boardList.map((board) => board.id)).toEqual([boardId])
    })

    it('rejects invalid data and keeps the current state', async () => {
      const store = useKanbanStore()
      await store.ensureLoaded()
      const before = store.boardList.map((board) => board.id)

      expect(store.importState({ boards: {}, columns: {} })).toBe(false)
      expect(store.boardList.map((board) => board.id)).toEqual(before)
    })
  })

  describe('resetToDemo', () => {
    it('discards changes and restores the seed', async () => {
      const { store, a } = await setup()

      store.resetToDemo()

      expect(store.state.columns[a.id]).toBeUndefined()
      expect(Object.keys(store.state.cards)).toHaveLength(6)
    })
  })
})
