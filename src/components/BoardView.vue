<script setup lang="ts">
import { ref } from 'vue'
import { createSeedState } from '@/seed'
import type { Card, Column, KanbanState } from '@/types'
import KanbanCard from './KanbanCard.vue'
import KanbanColumn from './KanbanColumn.vue'

// Phase 1: local state in a plain ref. Phase 2 moves this into a Pinia store.
const state = ref<KanbanState>(createSeedState())

const board = Object.values(state.value.boards)[0]

function columnsFor(columnIds: string[]): Column[] {
  return columnIds.flatMap((id) => state.value.columns[id] ?? [])
}

function cardsFor(column: Column): Card[] {
  return column.cardIds.flatMap((id) => state.value.cards[id] ?? [])
}
</script>

<template>
  <div v-if="board" class="board">
    <h1 class="board__title">{{ board.title }}</h1>

    <div class="board__columns">
      <KanbanColumn
        v-for="column in columnsFor(board.columnIds)"
        :key="column.id"
        :title="column.title"
        :card-count="column.cardIds.length"
      >
        <template #actions>
          <button
            class="board__column-action"
            type="button"
            disabled
            :aria-label="`Actions for ${column.title} (coming in phase 2)`"
            title="Column actions arrive in phase 2"
          >
            ⋯
          </button>
        </template>

        <KanbanCard v-for="card in cardsFor(column)" :key="card.id" :card="card" />
      </KanbanColumn>
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
}

.board__missing {
  padding: var(--space-6);
}
</style>
