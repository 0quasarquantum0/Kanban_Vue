<script setup lang="ts">
import { computed, nextTick, useTemplateRef } from 'vue'
import { useKanbanStore } from '@/stores/kanban'
import type { Card, Column, ID } from '@/types'

// The keyboard alternative to drag and drop: the same moves, as buttons.
const props = defineProps<{
  card: Card
  column: Column
  boardId: ID
  /** False while filters hide cards, when "up" and "down" would be ambiguous. */
  canReorder: boolean
}>()

const store = useKanbanStore()
const details = useTemplateRef('details')

const index = computed(() => props.column.cardIds.indexOf(props.card.id))
const otherColumns = computed(() =>
  store.columnsForBoard(props.boardId).filter((column) => column.id !== props.column.id),
)

async function move(toColumnId: ID, toIndex: number) {
  const cardId = props.card.id
  if (details.value) details.value.open = false
  store.moveCard(cardId, toColumnId, toIndex)

  // Moving to another column destroys this component and creates a new one there, so
  // keyboard focus would be lost. Wait for the DOM update, then focus the new menu.
  // (The old card is still in the DOM while its leave animation plays, hence the filter.)
  await nextTick()
  const menus = document.querySelectorAll<HTMLElement>(`[data-move-menu="${cardId}"]`)
  const current = [...menus].find((menu) => !menu.closest('[class*="-leave-active"]'))
  current?.focus()
}
</script>

<template>
  <!-- A native disclosure widget: the browser handles open/close and keyboard support. -->
  <details ref="details" class="move-menu">
    <summary
      class="move-menu__summary"
      :data-move-menu="card.id"
      :aria-label="`Move card: ${card.title}`"
    >
      Move to…
    </summary>
    <ul class="move-menu__list">
      <li v-for="target in otherColumns" :key="target.id">
        <button class="button" type="button" @click="move(target.id, Infinity)">
          {{ target.title }}
        </button>
      </li>
      <li>
        <button
          class="button"
          type="button"
          :disabled="!canReorder || index <= 0"
          @click="move(column.id, index - 1)"
        >
          ↑ Up
        </button>
      </li>
      <li>
        <button
          class="button"
          type="button"
          :disabled="!canReorder || index >= column.cardIds.length - 1"
          @click="move(column.id, index + 1)"
        >
          ↓ Down
        </button>
      </li>
    </ul>
  </details>
</template>

<style scoped>
.move-menu {
  margin-top: var(--space-2);
  font-size: 0.8125rem;
}

.move-menu__summary {
  display: inline-block;
  padding: 0 var(--space-1);
  color: var(--color-text-muted);
  border-radius: var(--radius);
  cursor: pointer;
}

.move-menu__summary:hover {
  color: var(--color-text);
  background: var(--color-hover);
}

.move-menu__list {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-1);
  margin: var(--space-2) 0 0;
  padding: 0;
  list-style: none;
}

.move-menu__list .button {
  padding: 0 var(--space-2);
  font-size: 0.8125rem;
}

.move-menu__list .button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
