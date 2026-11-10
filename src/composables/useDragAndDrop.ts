import { ref } from 'vue'
import type { ID } from '@/types'

interface DragAndDropOptions {
  /** Called once when a card is dropped. `toIndex` is the card's final index in the column. */
  onDrop: (cardId: ID, toColumnId: ID, toIndex: number) => void
}

/**
 * Card drag and drop on top of the native HTML5 events.
 *
 * Markup contract: each card element is `draggable` and carries `data-card-id`;
 * each column element is a drop zone that contains its card elements.
 */
export function useDragAndDrop(options: DragAndDropOptions) {
  const draggingCardId = ref<ID | null>(null)
  const overColumnId = ref<ID | null>(null)
  /** Where the card would land, counted among the column's cards without the dragged one. */
  const overIndex = ref<number | null>(null)

  function reset() {
    draggingCardId.value = null
    overColumnId.value = null
    overIndex.value = null
  }

  /** The drop index for the pointer position: the first card whose middle is below the pointer. */
  function indexAt(event: DragEvent): number {
    const zone = event.currentTarget
    if (!(zone instanceof HTMLElement)) return 0

    const cards = [...zone.querySelectorAll<HTMLElement>('[data-card-id]')].filter(
      (el) => el.dataset.cardId !== draggingCardId.value,
    )
    const index = cards.findIndex((el) => {
      const rect = el.getBoundingClientRect()
      return event.clientY < rect.top + rect.height / 2
    })
    return index === -1 ? cards.length : index
  }

  function onDragStart(event: DragEvent, cardId: ID) {
    draggingCardId.value = cardId
    if (event.dataTransfer) {
      event.dataTransfer.effectAllowed = 'move'
      // Firefox only starts a drag if some data is set.
      event.dataTransfer.setData('text/plain', cardId)
    }
  }

  function onDragOver(event: DragEvent, columnId: ID) {
    if (!draggingCardId.value) return
    if (event.dataTransfer) event.dataTransfer.dropEffect = 'move'
    overColumnId.value = columnId
    overIndex.value = indexAt(event)
  }

  function onDragLeave(event: DragEvent, columnId: ID) {
    // dragleave also fires when moving onto a child element; only react when leaving the column.
    const zone = event.currentTarget
    const next = event.relatedTarget
    if (zone instanceof Node && next instanceof Node && zone.contains(next)) return

    if (overColumnId.value === columnId) {
      overColumnId.value = null
      overIndex.value = null
    }
  }

  function onDrop(event: DragEvent, columnId: ID) {
    const cardId = draggingCardId.value
    if (cardId) options.onDrop(cardId, columnId, indexAt(event))
    reset()
  }

  /** Fires on the dragged card after every drag, including ones cancelled with Esc. */
  function onDragEnd() {
    reset()
  }

  /** Whether the drop indicator belongs before or after this card, or nowhere near it. */
  function dropPosition(columnId: ID, cardIds: ID[], cardId: ID): 'before' | 'after' | null {
    if (overColumnId.value !== columnId || overIndex.value === null) return null
    if (cardId === draggingCardId.value) return null

    const others = cardIds.filter((id) => id !== draggingCardId.value)
    const position = others.indexOf(cardId)
    if (position === overIndex.value) return 'before'
    if (position === others.length - 1 && overIndex.value === others.length) return 'after'
    return null
  }

  return {
    draggingCardId,
    overColumnId,
    overIndex,
    onDragStart,
    onDragOver,
    onDragLeave,
    onDrop,
    onDragEnd,
    dropPosition,
  }
}
