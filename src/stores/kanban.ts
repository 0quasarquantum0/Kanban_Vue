import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { createSeedState } from '@/seed'
import type { Board, Card, CardPatch, Column, ID, KanbanState } from '@/types'

export const useKanbanStore = defineStore('kanban', () => {
  // --- state ---
  const state = ref<KanbanState>(createSeedState())

  // --- getters ---
  // Until phase 6 adds routing, the app shows the first board.
  const firstBoard = computed<Board | undefined>(() => Object.values(state.value.boards)[0])

  // A computed can't take arguments, so these return a lookup function instead.
  const columnsForBoard = computed(() => (boardId: ID): Column[] => {
    const board = state.value.boards[boardId]
    return board ? board.columnIds.flatMap((id) => state.value.columns[id] ?? []) : []
  })

  const cardsForColumn = computed(() => (columnId: ID): Card[] => {
    const column = state.value.columns[columnId]
    return column ? column.cardIds.flatMap((id) => state.value.cards[id] ?? []) : []
  })

  function columnContaining(cardId: ID): Column | undefined {
    return Object.values(state.value.columns).find((column) => column.cardIds.includes(cardId))
  }

  // --- actions ---
  function addCard(columnId: ID, title: string): Card | undefined {
    const column = state.value.columns[columnId]
    if (!column) return undefined

    const card: Card = {
      id: crypto.randomUUID(),
      title,
      description: '',
      labels: [],
      dueDate: null,
      createdAt: new Date().toISOString(),
    }
    state.value.cards[card.id] = card
    column.cardIds.push(card.id)
    return card
  }

  function updateCard(cardId: ID, patch: CardPatch): void {
    const card = state.value.cards[cardId]
    if (card) Object.assign(card, patch)
  }

  function deleteCard(cardId: ID): void {
    const column = columnContaining(cardId)
    if (column) column.cardIds.splice(column.cardIds.indexOf(cardId), 1)
    delete state.value.cards[cardId]
  }

  function addColumn(boardId: ID, title: string): Column | undefined {
    const board = state.value.boards[boardId]
    if (!board) return undefined

    const column: Column = { id: crypto.randomUUID(), title, cardIds: [] }
    state.value.columns[column.id] = column
    board.columnIds.push(column.id)
    return column
  }

  function renameColumn(columnId: ID, title: string): void {
    const column = state.value.columns[columnId]
    if (column) column.title = title
  }

  /** Deletes the column and every card in it. */
  function deleteColumn(columnId: ID): void {
    const column = state.value.columns[columnId]
    if (!column) return

    for (const cardId of column.cardIds) delete state.value.cards[cardId]
    for (const board of Object.values(state.value.boards)) {
      const index = board.columnIds.indexOf(columnId)
      if (index !== -1) board.columnIds.splice(index, 1)
    }
    delete state.value.columns[columnId]
  }

  /**
   * Moves a card so that it ends up at `toIndex` in the target column.
   * `toIndex` is clamped to the valid range, so a large value appends.
   */
  function moveCard(cardId: ID, toColumnId: ID, toIndex: number): void {
    const from = columnContaining(cardId)
    const to = state.value.columns[toColumnId]
    if (!from || !to) return

    from.cardIds.splice(from.cardIds.indexOf(cardId), 1)
    const index = Math.max(0, Math.min(toIndex, to.cardIds.length))
    to.cardIds.splice(index, 0, cardId)
  }

  return {
    state,
    firstBoard,
    columnsForBoard,
    cardsForColumn,
    addCard,
    updateCard,
    deleteCard,
    addColumn,
    renameColumn,
    deleteColumn,
    moveCard,
  }
})
