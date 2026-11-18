import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import * as api from '@/api'
import { useServerSync } from '@/composables/useServerSync'
import { createSeedState } from '@/seed'
import type { Board, Card, CardPatch, Column, ID, KanbanState } from '@/types'
import { isKanbanState } from '@/validation'

export const useKanbanStore = defineStore('kanban', () => {
  // --- state ---
  // Empty until the first load from the Go server finishes (see useServerSync).
  const state = ref<KanbanState>({ boards: {}, columns: {}, cards: {} })
  const sync = useServerSync(state)

  // --- getters ---
  const boardList = computed<Board[]>(() => Object.values(state.value.boards))

  // A computed can't take arguments, so these return a lookup function instead.
  const columnsForBoard = computed(() => (boardId: ID): Column[] => {
    const board = state.value.boards[boardId]
    return board ? board.columnIds.flatMap((id) => state.value.columns[id] ?? []) : []
  })

  const cardsForColumn = computed(() => (columnId: ID): Card[] => {
    const column = state.value.columns[columnId]
    return column ? column.cardIds.flatMap((id) => state.value.cards[id] ?? []) : []
  })

  /** Every distinct label used on the board, sorted. */
  const labelsForBoard = computed(() => (boardId: ID): string[] => {
    const cards = columnsForBoard
      .value(boardId)
      .flatMap((column) => cardsForColumn.value(column.id))
    return [...new Set(cards.flatMap((card) => card.labels))].sort()
  })

  function columnContaining(cardId: ID): Column | undefined {
    return Object.values(state.value.columns).find((column) => column.cardIds.includes(cardId))
  }

  // --- actions ---
  // Each action changes `state` first, so the screen updates at once, and then queues the
  // matching request for the server. An action on something that does not exist does nothing.

  function addBoard(title: string): Board {
    const board: Board = { id: crypto.randomUUID(), title, columnIds: [] }
    state.value.boards[board.id] = board
    sync.enqueue(() => api.addBoard(board.id, title), board.id)
    return board
  }

  function renameBoard(boardId: ID, title: string): void {
    const board = state.value.boards[boardId]
    if (!board) return
    board.title = title
    sync.enqueue(() => api.renameBoard(boardId, title))
  }

  /** Deletes the board, its columns, and their cards. The server does the same in one request. */
  function deleteBoard(boardId: ID): void {
    const board = state.value.boards[boardId]
    if (!board) return

    // Copy first: removeColumn splices this same array.
    for (const columnId of board.columnIds.slice()) removeColumn(columnId)
    delete state.value.boards[boardId]
    sync.enqueue(() => api.deleteBoard(boardId))
  }

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
    sync.enqueue(() => api.addCard(columnId, card.id, title), card.id)
    return card
  }

  function updateCard(cardId: ID, patch: CardPatch): void {
    const card = state.value.cards[cardId]
    if (!card) return
    Object.assign(card, patch)
    sync.enqueue(() => api.updateCard(cardId, patch))
  }

  function deleteCard(cardId: ID): void {
    if (!state.value.cards[cardId]) return
    const column = columnContaining(cardId)
    if (column) column.cardIds.splice(column.cardIds.indexOf(cardId), 1)
    delete state.value.cards[cardId]
    sync.enqueue(() => api.deleteCard(cardId))
  }

  function addColumn(boardId: ID, title: string): Column | undefined {
    const board = state.value.boards[boardId]
    if (!board) return undefined

    const column: Column = { id: crypto.randomUUID(), title, cardIds: [] }
    state.value.columns[column.id] = column
    board.columnIds.push(column.id)
    sync.enqueue(() => api.addColumn(boardId, column.id, title), column.id)
    return column
  }

  function renameColumn(columnId: ID, title: string): void {
    const column = state.value.columns[columnId]
    if (!column) return
    column.title = title
    sync.enqueue(() => api.renameColumn(columnId, title))
  }

  /** Deletes the column and every card in it. */
  function deleteColumn(columnId: ID): void {
    if (!state.value.columns[columnId]) return
    removeColumn(columnId)
    sync.enqueue(() => api.deleteColumn(columnId))
  }

  /** The local half of deleting a column, shared by deleteColumn and deleteBoard. */
  function removeColumn(columnId: ID): void {
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
    sync.enqueue(() => api.moveCard(cardId, toColumnId, index))
  }

  /** Replaces all state with `data` if it is a valid KanbanState. Returns whether it was. */
  function importState(data: unknown): boolean {
    if (!isKanbanState(data)) return false
    replaceState(data)
    return true
  }

  function resetToDemo(): void {
    replaceState(createSeedState())
  }

  /** Whole-state writes carry the version they are based on, so another tab can't be overwritten. */
  function replaceState(next: KanbanState): void {
    state.value = next
    sync.enqueue(() => api.putState(sync.version.value, next))
  }

  return {
    state,
    ...sync,
    boardList,
    columnsForBoard,
    cardsForColumn,
    labelsForBoard,
    addBoard,
    renameBoard,
    deleteBoard,
    addCard,
    updateCard,
    deleteCard,
    addColumn,
    renameColumn,
    deleteColumn,
    moveCard,
    importState,
    resetToDemo,
  }
})
