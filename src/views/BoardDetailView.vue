<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, RouterView, useRoute } from 'vue-router'
import BoardView from '@/components/BoardView.vue'
import { useKanbanStore } from '@/stores/kanban'
import type { ID } from '@/types'

// Filled from the `:boardId` route param by the route's `props` option.
const props = defineProps<{
  boardId: ID
}>()

const store = useKanbanStore()
const route = useRoute()

// The route guard checks the board on entry, but it can still disappear afterwards
// (deleted in another tab, for example), so the template handles `undefined` too.
const board = computed(() => store.state.boards[props.boardId])
</script>

<template>
  <template v-if="board">
    <BoardView :board="board" />
    <!--
      The nested route (the card editor) renders here. The key gives each card its own
      component instance, so going back/forward between two cards doesn't reuse stale form state.
    -->
    <RouterView :key="route.path" />
  </template>
  <main v-else class="missing">
    <p>This board no longer exists.</p>
    <RouterLink :to="{ name: 'boards' }">Back to all boards</RouterLink>
  </main>
</template>

<style scoped>
.missing {
  padding: var(--space-6);
}
</style>
