export type ID = string // crypto.randomUUID()

export interface Card {
  id: ID
  title: string
  description: string
  labels: string[] // e.g. ['bug', 'ui']
  dueDate: string | null // ISO date
  createdAt: string
}

/** The fields of a card that can be edited after creation. */
export type CardPatch = Partial<Pick<Card, 'title' | 'description' | 'labels' | 'dueDate'>>

export interface Column {
  id: ID
  title: string
  cardIds: ID[] // order matters
}

export interface Board {
  id: ID
  title: string
  columnIds: ID[] // order matters
}

export interface KanbanState {
  boards: Record<ID, Board>
  columns: Record<ID, Column>
  cards: Record<ID, Card>
}
