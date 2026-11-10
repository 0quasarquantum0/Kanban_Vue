<script setup lang="ts">
import InlineEdit from './InlineEdit.vue'

// A named model: the parent binds it with v-model:title.
const title = defineModel<string>('title', { required: true })

defineProps<{
  cardCount: number
}>()
</script>

<template>
  <section class="column">
    <header class="column__header">
      <h2 class="column__title">
        <InlineEdit v-model="title" label="Column title" />
      </h2>
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

    <footer class="column__footer">
      <slot name="footer" />
    </footer>
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
  margin-bottom: var(--space-1);
}

.column__title {
  flex: 1;
  min-width: 0;
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

.column__cards {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  /* Room for the drop indicator above the first card and below the last. */
  padding-block: var(--space-2);
  overflow-y: auto;
}

.column__empty {
  margin: var(--space-2) 0;
  padding: var(--space-4) var(--space-2);
  font-size: 0.8125rem;
  text-align: center;
  color: var(--color-text-muted);
  border: 1px dashed var(--color-border);
  border-radius: var(--radius);
}
</style>
