<script setup lang="ts">
import { computed } from 'vue'
import { useBoardFilters } from '@/composables/useBoardFilters'
import { useDragAndDrop } from '@/composables/useDragAndDrop'
import { useKanbanStore } from '@/stores/kanban'
import type { Board, Column } from '@/types'
import AddItemForm from './AddItemForm.vue'
import BoardHeader from './BoardHeader.vue'
import FilterBar from './FilterBar.vue'
import KanbanCard from './KanbanCard.vue'
import KanbanColumn from './KanbanColumn.vue'

const props = defineProps<{
  board: Board
}>()

const store = useKanbanStore()
const { isFiltering, matches } = useBoardFilters()

// The filtered view is derived, never stored: it recomputes when the board, its cards or
// the filters change, and the store keeps the full lists.
const columns = computed(() =>
  store.columnsForBoard(props.board.id).map((column) => {
    const cards = store.cardsForColumn(column.id)
    return { column, cards: cards.filter(matches.value), total: cards.length }
  }),
)

// Refs are destructured to the top level so the template unwraps them automatically.
const {
  draggingCardId,
  overColumnId,
  onDragStart,
  onDragOver,
  onDragLeave,
  onDrop,
  onDragEnd,
  dropPosition,
} = useDragAndDrop({ onDrop: store.moveCard })

function removeColumn(column: Column) {
  const count = column.cardIds.length
  const message = `Delete "${column.title}" and its ${count} ${count === 1 ? 'card' : 'cards'}?`
  if (count === 0 || window.confirm(message)) store.deleteColumn(column.id)
}
</script>

<template>
  <div class="board">
    <!-- `board` is a prop, so its title can't be v-modelled; this is v-model written out by hand. -->
    <BoardHeader :title="board.title" @update:title="store.renameBoard(board.id, $event)" />
    <FilterBar :labels="store.labelsForBoard(board.id)" />

    <div class="board__columns">
      <KanbanColumn
        v-for="{ column, cards, total } in columns"
        :key="column.id"
        v-model:title="column.title"
        :card-count="cards.length"
        :total-count="isFiltering ? total : undefined"
        :empty-text="isFiltering && total > 0 ? 'No matching cards' : undefined"
        :class="{ 'is-drop-target': overColumnId === column.id }"
        @dragover.prevent="onDragOver($event, column.id)"
        @dragleave="onDragLeave($event, column.id)"
        @drop.prevent="onDrop($event, column.id)"
      >
        <template #actions>
          <button
            class="icon-button icon-button--danger"
            type="button"
            :aria-label="`Delete column: ${column.title}`"
            @click="removeColumn(column)"
          >
            ×
          </button>
        </template>

        <KanbanCard
          v-for="card in cards"
          :key="card.id"
          :card="card"
          :draggable="!isFiltering"
          :data-card-id="card.id"
          :data-drop="dropPosition(column.id, column.cardIds, card.id)"
          :class="{ 'is-dragging': draggingCardId === card.id }"
          @dragstart="onDragStart($event, card.id)"
          @dragend="onDragEnd"
          @delete="store.deleteCard(card.id)"
        />

        <template #footer>
          <AddItemForm
            label="Add card"
            placeholder="Card title"
            @add="store.addCard(column.id, $event)"
          />
        </template>
      </KanbanColumn>

      <div class="board__add-column">
        <AddItemForm
          label="Add column"
          placeholder="Column title"
          @add="store.addColumn(board.id, $event)"
        />
      </div>
    </div>
  </div>
</template>

<style scoped>
.board {
  display: flex;
  flex-direction: column;
  height: 100vh;
  padding: var(--space-4) var(--space-6);
}

.board__columns {
  display: flex;
  flex: 1;
  align-items: flex-start;
  gap: var(--space-4);
  min-height: 0;
  /* Tall columns scroll the whole board, not themselves. */
  overflow: auto;
}

.board__add-column {
  flex: 0 0 18rem;
  padding: var(--space-2);
  background: var(--color-column);
  border-radius: var(--radius);
}
</style>
