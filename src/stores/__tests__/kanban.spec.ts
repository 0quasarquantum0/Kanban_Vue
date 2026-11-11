import { beforeEach, describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import { createPinia, setActivePinia } from 'pinia'
import { createSeedState } from '@/seed'
import { STORAGE_KEY, useKanbanStore } from '../kanban'

/** A store with two fresh, empty columns on the seed board. */
function setup() {
  const store = useKanbanStore()
  const board = store.firstBoard
  if (!board) throw new Error('seed has no board')

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
    localStorage.clear()
    setActivePinia(createPinia())
  })

  describe('addCard', () => {
    it('appends a card with defaults to the column', () => {
      const { store, a, add, titles } = setup()
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

    it('does nothing for an unknown column', () => {
      const { store } = setup()
      const before = Object.keys(store.state.cards).length

      expect(store.addCard('nope', 'x')).toBeUndefined()
      expect(Object.keys(store.state.cards)).toHaveLength(before)
    })
  })

  describe('moveCard', () => {
    it('reorders within a column', () => {
      const { store, a, add, titles } = setup()
      const one = add(a.id, '1')
      add(a.id, '2')
      add(a.id, '3')

      store.moveCard(one, a.id, 2)
      expect(titles(a.id)).toEqual(['2', '3', '1'])

      store.moveCard(one, a.id, 0)
      expect(titles(a.id)).toEqual(['1', '2', '3'])
    })

    it('moves between columns at the given index', () => {
      const { store, a, b, add, titles } = setup()
      const one = add(a.id, '1')
      add(a.id, '2')
      add(b.id, 'x')
      add(b.id, 'y')

      store.moveCard(one, b.id, 1)

      expect(titles(a.id)).toEqual(['2'])
      expect(titles(b.id)).toEqual(['x', '1', 'y'])
    })

    it('clamps an out-of-range index', () => {
      const { store, a, b, add, titles } = setup()
      const one = add(a.id, '1')
      add(b.id, 'x')

      store.moveCard(one, b.id, 99)
      expect(titles(b.id)).toEqual(['x', '1'])

      store.moveCard(one, b.id, -5)
      expect(titles(b.id)).toEqual(['1', 'x'])
    })

    it('leaves the card in place for an unknown target column', () => {
      const { store, a, add, titles } = setup()
      const one = add(a.id, '1')

      store.moveCard(one, 'nope', 0)
      expect(titles(a.id)).toEqual(['1'])
    })
  })

  describe('deleteColumn', () => {
    it('removes the column, its cards, and its place on the board', () => {
      const { store, board, a, b, add } = setup()
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

  describe('persistence', () => {
    it('saves changes to localStorage and loads them in a fresh store', async () => {
      const { a, add } = setup()
      add(a.id, 'persisted')
      await nextTick() // useStorage writes after Vue's next flush

      // A new Pinia instance stands in for a page refresh.
      setActivePinia(createPinia())
      const reloaded = useKanbanStore()

      expect(reloaded.cardsForColumn(a.id).map((card) => card.title)).toEqual(['persisted'])
    })

    it('falls back to the seed when stored data is invalid', () => {
      localStorage.setItem(STORAGE_KEY, JSON.stringify({ boards: 'nope' }))
      const store = useKanbanStore()

      expect(store.firstBoard?.title).toBe('Vue Kanban')
    })
  })

  describe('importState', () => {
    it('replaces the state with valid data', () => {
      const store = useKanbanStore()
      const next = createSeedState()
      const boardId = Object.keys(next.boards)[0]

      expect(store.importState(next)).toBe(true)
      expect(store.firstBoard?.id).toBe(boardId)
    })

    it('rejects invalid data and keeps the current state', () => {
      const store = useKanbanStore()
      const before = store.firstBoard?.id

      expect(store.importState({ boards: {}, columns: {} })).toBe(false)
      expect(store.firstBoard?.id).toBe(before)
    })
  })

  describe('resetToDemo', () => {
    it('discards changes and restores the seed', () => {
      const { store, a } = setup()

      store.resetToDemo()

      expect(store.state.columns[a.id]).toBeUndefined()
      expect(Object.keys(store.state.cards)).toHaveLength(6)
    })
  })
})
