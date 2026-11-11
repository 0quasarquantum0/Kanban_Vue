import { describe, expect, it } from 'vitest'
import { createSeedState } from '@/seed'
import type { KanbanState } from '@/types'
import { isKanbanState } from '@/validation'

/** The seed after a JSON round trip, as it would arrive from an imported file. */
function seed(): KanbanState {
  return JSON.parse(JSON.stringify(createSeedState())) as KanbanState
}

function firstColumn(state: KanbanState) {
  const column = Object.values(state.columns)[0]
  if (!column) throw new Error('seed has no column')
  return column
}

describe('isKanbanState', () => {
  it('accepts the seed and an empty state', () => {
    expect(isKanbanState(seed())).toBe(true)
    expect(isKanbanState({ boards: {}, columns: {}, cards: {} })).toBe(true)
  })

  it.each([null, 'text', 42, [], {}, { boards: {}, columns: {} }, { boards: [], columns: {}, cards: {} }])(
    'rejects %j',
    (value) => {
      expect(isKanbanState(value)).toBe(false)
    },
  )

  it('rejects a card with a wrongly typed field', () => {
    const state = seed()
    const cardId = Object.keys(state.cards)[0] ?? ''
    Object.assign(state.cards[cardId] ?? {}, { labels: 'bug' })

    expect(isKanbanState(state)).toBe(false)
  })

  it('rejects an entity stored under a different id', () => {
    const state = seed()
    const column = firstColumn(state)
    delete state.columns[column.id]
    state.columns['other-key'] = column

    expect(isKanbanState(state)).toBe(false)
  })

  it('rejects a column that points at a missing card', () => {
    const state = seed()
    firstColumn(state).cardIds.push('missing')

    expect(isKanbanState(state)).toBe(false)
  })

  it('rejects a card placed in two columns', () => {
    const state = seed()
    const [first, second] = Object.values(state.columns)
    const cardId = first?.cardIds[0]
    if (!second || !cardId) throw new Error('seed too small')
    second.cardIds.push(cardId)

    expect(isKanbanState(state)).toBe(false)
  })
})
