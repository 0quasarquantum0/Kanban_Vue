<script setup lang="ts">
import { RouterLink } from 'vue-router'
import InlineEdit from './InlineEdit.vue'

const title = defineModel<string>('title', { required: true })
</script>

<template>
  <header class="board-header">
    <RouterLink class="board-header__back" :to="{ name: 'boards' }">← All boards</RouterLink>
    <h1 class="board-header__title">
      <InlineEdit v-model="title" label="Board title" />
    </h1>
  </header>
</template>

<style scoped>
.board-header {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  margin-bottom: var(--space-3);
}

.board-header__back {
  font-size: 0.875rem;
  color: var(--color-text-muted);
  text-decoration: none;
}

.board-header__back:hover {
  color: var(--color-accent);
}

.board-header__title {
  align-self: stretch;
}
</style>
