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

    <!--
      Default slot: the parent passes the cards in as children.
      TransitionGroup renders the wrapping div and animates its keyed children as they
      enter, leave and change position. The empty message is one of those children.
    -->
    <TransitionGroup tag="div" class="column__cards" name="card-list">
      <slot />
      <p v-if="cardCount === 0" key="empty" class="column__empty">No cards yet</p>
    </TransitionGroup>

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
  /* Clip cards while they animate so they never poke out of the column or cause a scrollbar. */
  overflow: hidden;
  /* Positioning context for cards that are animating out. */
  position: relative;
}

/*
  The cards come from the parent through the slot, so they don't carry this component's
  scope id. :deep() lets these rules reach them.
*/
.column__cards :deep(.card-list-enter-active),
.column__cards :deep(.card-list-leave-active) {
  transition:
    opacity 0.2s ease,
    transform 0.2s ease;
}

.column__cards :deep(.card-list-enter-from),
.column__cards :deep(.card-list-leave-to) {
  opacity: 0;
  transform: scale(0.96);
}

/* Applied to siblings whose position changed; Vue animates them from old to new place. */
.column__cards :deep(.card-list-move) {
  transition: transform 0.25s ease;
}

/* Take leaving items out of the flow so the siblings below can slide up straight away. */
.column__cards :deep(.card-list-leave-active) {
  position: absolute;
  inset-inline: 0;
}

@media (prefers-reduced-motion: reduce) {
  .column__cards :deep(.card-list-enter-active),
  .column__cards :deep(.card-list-leave-active),
  .column__cards :deep(.card-list-move) {
    transition: none;
  }
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
