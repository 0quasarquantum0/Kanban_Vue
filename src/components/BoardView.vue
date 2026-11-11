<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { useDragAndDrop } from '@/composables/useDragAndDrop'
import { useKanbanStore } from '@/stores/kanban'
import type { Board, Column } from '@/types'
import AddItemForm from './AddItemForm.vue'
import InlineEdit from './InlineEdit.vue'
import KanbanCard from './KanbanCard.vue'
import KanbanColumn from './KanbanColumn.vue'

defineProps<{
  board: Board
}>()

const store = useKanbanStore()

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
    <header class="board__header">
      <RouterLink class="board__back" :to="{ name: 'boards' }">← All boards</RouterLink>
      <h1 class="board__title">
        <InlineEdit
          :model-value="board.title"
          label="Board title"
          @update:model-value="store.renameBoard(board.id, $event)"
        />
      </h1>
    </header>

    <div class="board__columns">
      <KanbanColumn
        v-for="column in store.columnsForBoard(board.id)"
        :key="column.id"
        v-model:title="column.title"
        :card-count="column.cardIds.length"
        :class="{ 'is-drop-target': overColumnId === column.id }"
        @dragover.prevent="onDragOver($event, column.id)"
        @dragleave="onDragLeave($event, column.id)"
        @drop.prevent="onDrop($event, column.id)"
      >
        <template #actions>
          <button
            class="board__column-action"
            type="button"
            :aria-label="`Delete column: ${column.title}`"
            @click="removeColumn(column)"
          >
            ×
          </button>
        </template>

        <KanbanCard
          v-for="card in store.cardsForColumn(column.id)"
          :key="card.id"
          :card="card"
          draggable="true"
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

.board__header {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  margin-bottom: var(--space-4);
}

.board__back {
  font-size: 0.875rem;
  color: var(--color-text-muted);
  text-decoration: none;
}

.board__back:hover {
  color: var(--color-accent);
}

.board__title {
  align-self: stretch;
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

.board__column-action {
  padding: 0 var(--space-2);
  font-size: 1rem;
  line-height: 1.5;
  color: var(--color-text-muted);
  background: none;
  border: none;
  border-radius: var(--radius);
  cursor: pointer;
}

.board__column-action:hover {
  color: var(--color-danger);
  background: var(--color-hover);
}

.board__add-column {
  flex: 0 0 18rem;
  padding: var(--space-2);
  background: var(--color-column);
  border-radius: var(--radius);
}
</style>
