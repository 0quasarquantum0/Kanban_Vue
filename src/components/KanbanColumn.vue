<script setup lang="ts">
defineProps<{
  title: string
  cardCount: number
}>()
</script>

<template>
  <section class="column">
    <header class="column__header">
      <h2 class="column__title">{{ title }}</h2>
      <span class="column__count">{{ cardCount }}</span>
      <div class="column__actions">
        <!-- Named slot: the parent decides which buttons appear here. -->
        <slot name="actions" />
      </div>
    </header>

    <!-- Default slot: the parent passes the cards in as children. -->
    <div v-if="cardCount > 0" class="column__cards">
      <slot />
    </div>
    <p v-else class="column__empty">No cards yet</p>
  </section>
</template>

<style scoped>
.column {
  display: flex;
  flex-direction: column;
  flex: 0 0 18rem;
  max-height: 100%;
  padding: var(--space-3);
  background: var(--color-column);
  border-radius: var(--radius);
}

.column__header {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin-bottom: var(--space-3);
}

.column__title {
  margin: 0;
  font-size: 0.875rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.column__count {
  font-size: 0.75rem;
  color: var(--color-text-muted);
}

.column__actions {
  margin-left: auto;
}

.column__cards {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  overflow-y: auto;
}

.column__empty {
  margin: 0;
  padding: var(--space-4) var(--space-2);
  font-size: 0.8125rem;
  text-align: center;
  color: var(--color-text-muted);
  border: 1px dashed var(--color-border);
  border-radius: var(--radius);
}
</style>
