<script setup lang="ts">
import { RouterLink } from 'vue-router'
import InlineEdit from './InlineEdit.vue'
import ThemeToggle from './ThemeToggle.vue'

const title = defineModel<string>('title', { required: true })
</script>

<template>
  <header class="board-header">
    <div class="board-header__top">
      <RouterLink class="board-header__back" :to="{ name: 'boards' }">← All boards</RouterLink>
      <ThemeToggle />
    </div>
    <h1>
      <InlineEdit v-model="title" label="Board title" />
    </h1>
  </header>
</template>

<style scoped>
.board-header {
  margin-bottom: var(--space-3);
}

.board-header__top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
}

.board-header__back {
  font-size: 0.875rem;
  color: var(--color-text-muted);
  text-decoration: none;
}

.board-header__back:hover {
  color: var(--color-accent);
}
</style>
