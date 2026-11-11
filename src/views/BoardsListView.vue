<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { RouterLink } from 'vue-router'
import AddItemForm from '@/components/AddItemForm.vue'
import BoardToolbar from '@/components/BoardToolbar.vue'
import InlineEdit from '@/components/InlineEdit.vue'
import { useKanbanStore } from '@/stores/kanban'
import type { Board } from '@/types'

const store = useKanbanStore()
const { boardList } = storeToRefs(store)

function summary(board: Board): string {
  const columns = store.columnsForBoard(board.id)
  const cards = columns.reduce((total, column) => total + column.cardIds.length, 0)
  const plural = (count: number, word: string) => `${count} ${word}${count === 1 ? '' : 's'}`
  return `${plural(columns.length, 'column')} · ${plural(cards, 'card')}`
}

function removeBoard(board: Board) {
  const isEmpty = board.columnIds.length === 0
  if (isEmpty || window.confirm(`Delete "${board.title}" and everything on it?`)) {
    store.deleteBoard(board.id)
  }
}
</script>

<template>
  <main class="boards">
    <header class="boards__header">
      <h1>Kanban</h1>
      <BoardToolbar />
    </header>

    <ul class="boards__list">
      <li v-for="board in boardList" :key="board.id" class="boards__item">
        <div class="boards__item-header">
          <h2 class="boards__title">
            <InlineEdit v-model="board.title" label="Board title" />
          </h2>
          <button
            class="icon-button icon-button--danger"
            type="button"
            :aria-label="`Delete board: ${board.title}`"
            @click="removeBoard(board)"
          >
            ×
          </button>
        </div>
        <p class="boards__summary">{{ summary(board) }}</p>
        <RouterLink class="boards__open" :to="{ name: 'board', params: { boardId: board.id } }">
          Open board →
        </RouterLink>
      </li>

      <li class="boards__item boards__item--add">
        <AddItemForm label="Add board" placeholder="Board title" @add="store.addBoard($event)" />
      </li>
    </ul>

    <p v-if="boardList.length === 0" class="boards__empty">
      No boards yet. Add one to get started.
    </p>
  </main>
</template>

<style scoped>
.boards {
  padding: var(--space-4) var(--space-6);
}

.boards__header {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2) var(--space-4);
  margin-bottom: var(--space-4);
}

.boards__list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(16rem, 1fr));
  gap: var(--space-4);
  margin: 0;
  padding: 0;
  list-style: none;
}

.boards__item {
  padding: var(--space-3);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  box-shadow: var(--shadow-card);
}

.boards__item--add {
  align-self: start;
  padding: var(--space-2);
  background: var(--color-column);
  border: none;
  box-shadow: none;
}

.boards__item-header {
  display: flex;
  align-items: flex-start;
  gap: var(--space-1);
}

.boards__title {
  flex: 1;
  min-width: 0;
  margin: 0;
  font-size: 1.0625rem;
}

.boards__summary {
  margin: var(--space-1) var(--space-1) var(--space-3);
  font-size: 0.8125rem;
  color: var(--color-text-muted);
}

.boards__open {
  margin-left: var(--space-1);
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--color-accent);
  text-decoration: none;
}

.boards__open:hover {
  text-decoration: underline;
}

.boards__empty {
  margin-top: var(--space-4);
  color: var(--color-text-muted);
}
</style>
