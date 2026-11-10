<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { useKanbanStore } from '@/stores/kanban'
import type { Column } from '@/types'
import AddItemForm from './AddItemForm.vue'
import KanbanCard from './KanbanCard.vue'
import KanbanColumn from './KanbanColumn.vue'

const store = useKanbanStore()
// storeToRefs keeps `board` reactive after destructuring. Actions are called on `store` directly.
const { firstBoard: board } = storeToRefs(store)

function removeColumn(column: Column) {
  const count = column.cardIds.length
  const message = `Delete "${column.title}" and its ${count} ${count === 1 ? 'card' : 'cards'}?`
  if (count === 0 || window.confirm(message)) store.deleteColumn(column.id)
}
</script>

<template>
  <div v-if="board" class="board">
    <h1 class="board__title">{{ board.title }}</h1>

    <div class="board__columns">
      <KanbanColumn
        v-for="column in store.columnsForBoard(board.id)"
        :key="column.id"
        v-model:title="column.title"
        :card-count="column.cardIds.length"
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
          @update="store.updateCard(card.id, $event)"
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
  <p v-else class="board__missing">No board found.</p>
</template>

<style scoped>
.board {
  display: flex;
  flex-direction: column;
  height: 100vh;
  padding: var(--space-4) var(--space-6);
}

.board__title {
  margin-bottom: var(--space-4);
}

.board__columns {
  display: flex;
  flex: 1;
  align-items: flex-start;
  gap: var(--space-4);
  min-height: 0;
  overflow-x: auto;
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

.board__missing {
  padding: var(--space-6);
}
</style>
