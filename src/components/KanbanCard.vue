<script setup lang="ts">
import { RouterLink, useRoute } from 'vue-router'
import type { Card } from '@/types'

defineProps<{
  card: Card
}>()

const emit = defineEmits<{
  delete: []
}>()

const route = useRoute()

// Date-only ISO strings parse as UTC midnight, so format in UTC to avoid an off-by-one day.
const dateFormat = new Intl.DateTimeFormat(undefined, {
  day: 'numeric',
  month: 'short',
  timeZone: 'UTC',
})

function formatDate(iso: string): string {
  return dateFormat.format(new Date(iso))
}
</script>

<template>
  <article class="card">
    <header class="card__header">
      <h3 class="card__title">
        <!--
          Opens the editor route. `boardId` isn't passed: for a named route, the router
          reuses matching params from the current location. The query is passed along so
          the board's filters stay in the URL.
          draggable="false" stops the browser dragging the link instead of the card.
        -->
        <RouterLink
          class="card__link"
          :to="{ name: 'card', params: { cardId: card.id }, query: route.query }"
          draggable="false"
        >
          {{ card.title }}
        </RouterLink>
      </h3>
      <button
        class="icon-button icon-button--danger"
        type="button"
        :aria-label="`Delete card: ${card.title}`"
        @click="emit('delete')"
      >
        ×
      </button>
    </header>

    <p v-if="card.description" class="card__description">{{ card.description }}</p>

    <footer v-if="card.labels.length || card.dueDate" class="card__meta">
      <ul v-if="card.labels.length" class="card__labels">
        <li v-for="label in card.labels" :key="label" class="card__label">{{ label }}</li>
      </ul>
      <time v-if="card.dueDate" class="card__due" :datetime="card.dueDate">
        Due {{ formatDate(card.dueDate) }}
      </time>
    </footer>

    <!-- Extra controls from the parent, e.g. the "Move to…" menu. -->
    <slot />
  </article>
</template>

<style scoped>
.card {
  padding: var(--space-3);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  box-shadow: var(--shadow-card);
}

.card[draggable='true'] {
  cursor: grab;
}

.card__header {
  display: flex;
  align-items: flex-start;
  gap: var(--space-1);
}

.card__title {
  flex: 1;
  min-width: 0;
  margin: 0;
  font-size: 0.9375rem;
  font-weight: 600;
  overflow-wrap: anywhere;
}

.card__link {
  color: inherit;
  text-decoration: none;
}

.card__link:hover {
  color: var(--color-accent);
  text-decoration: underline;
}

.card__description {
  margin: var(--space-1) 0 0;
  font-size: 0.8125rem;
  color: var(--color-text-muted);
  overflow-wrap: anywhere;
}

.card__meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
  margin-top: var(--space-3);
}

.card__labels {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-1);
  margin: 0;
  padding: 0;
  list-style: none;
}

.card__label {
  padding: 0 var(--space-2);
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--color-label-text);
  background: var(--color-label-bg);
  border-radius: 999px;
}

.card__due {
  margin-left: auto;
  font-size: 0.75rem;
  color: var(--color-text-muted);
}
</style>
