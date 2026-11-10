import type { Card, Column, KanbanState } from './types'

type CardSeed = Pick<Card, 'title'> & Partial<Pick<Card, 'description' | 'labels' | 'dueDate'>>

const columnSeeds: { title: string; cards: CardSeed[] }[] = [
  {
    title: 'To do',
    cards: [
      {
        title: 'Drag and drop between columns',
        description: 'Native HTML5 drag events wrapped in a composable.',
        labels: ['feature'],
        dueDate: '2026-11-20',
      },
      {
        title: 'Persist the board to localStorage',
        labels: ['feature'],
      },
      {
        title: 'Card title overflows on narrow screens',
        description: 'Long words without spaces push the card wider than its column.',
        labels: ['bug', 'ui'],
        dueDate: '2026-11-14',
      },
    ],
  },
  {
    title: 'Doing',
    cards: [
      {
        title: 'Build the static board',
        description: 'BoardView, KanbanColumn and KanbanCard with props and slots.',
        labels: ['ui'],
        dueDate: '2026-11-12',
      },
      {
        title: 'Read the Vue docs on reactivity',
        labels: ['learning'],
      },
    ],
  },
  {
    title: 'Done',
    cards: [
      {
        title: 'Scaffold the project',
        description: 'create-vue with TypeScript, Router, Pinia and Vitest.',
        labels: ['chore'],
      },
    ],
  },
]

/** Builds a fresh demo state with new IDs on every call. */
export function createSeedState(): KanbanState {
  const state: KanbanState = { boards: {}, columns: {}, cards: {} }
  const createdAt = new Date().toISOString()

  const columnIds = columnSeeds.map(({ title, cards }) => {
    const cardIds = cards.map((seed) => {
      const card: Card = {
        id: crypto.randomUUID(),
        description: '',
        labels: [],
        dueDate: null,
        createdAt,
        ...seed,
      }
      state.cards[card.id] = card
      return card.id
    })

    const column: Column = { id: crypto.randomUUID(), title, cardIds }
    state.columns[column.id] = column
    return column.id
  })

  const boardId = crypto.randomUUID()
  state.boards[boardId] = { id: boardId, title: 'Vue Kanban', columnIds }

  return state
}
