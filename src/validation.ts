import type { Board, Card, Column, KanbanState } from './types'

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function isStringArray(value: unknown): value is string[] {
  return Array.isArray(value) && value.every((item) => typeof item === 'string')
}

function isCard(value: unknown): value is Card {
  return (
    isRecord(value) &&
    typeof value.id === 'string' &&
    typeof value.title === 'string' &&
    typeof value.description === 'string' &&
    isStringArray(value.labels) &&
    (value.dueDate === null || typeof value.dueDate === 'string') &&
    typeof value.createdAt === 'string'
  )
}

function isColumn(value: unknown): value is Column {
  return (
    isRecord(value) &&
    typeof value.id === 'string' &&
    typeof value.title === 'string' &&
    isStringArray(value.cardIds)
  )
}

function isBoard(value: unknown): value is Board {
  return (
    isRecord(value) &&
    typeof value.id === 'string' &&
    typeof value.title === 'string' &&
    isStringArray(value.columnIds)
  )
}

/** A record whose values all pass `isItem` and are stored under their own id. */
function isEntityRecord<T extends { id: string }>(
  value: unknown,
  isItem: (item: unknown) => item is T,
): value is Record<string, T> {
  return (
    isRecord(value) && Object.entries(value).every(([id, item]) => isItem(item) && item.id === id)
  )
}

/** Every id exists in `entities`, and no id is referenced twice. */
function referencesAreValid(ids: string[], entities: Record<string, unknown>): boolean {
  const known = new Set(Object.keys(entities))
  return new Set(ids).size === ids.length && ids.every((id) => known.has(id))
}

/**
 * Checks shape and referential integrity: every column on a board and every card in a
 * column must exist, and each can only be placed once.
 */
export function isKanbanState(value: unknown): value is KanbanState {
  if (!isRecord(value)) return false
  const { boards, columns, cards } = value

  if (
    !isEntityRecord(boards, isBoard) ||
    !isEntityRecord(columns, isColumn) ||
    !isEntityRecord(cards, isCard)
  ) {
    return false
  }

  const columnIds = Object.values(boards).flatMap((board) => board.columnIds)
  const cardIds = Object.values(columns).flatMap((column) => column.cardIds)
  return referencesAreValid(columnIds, columns) && referencesAreValid(cardIds, cards)
}
