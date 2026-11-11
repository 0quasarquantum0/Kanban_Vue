<script setup lang="ts">
import { computed, ref, useTemplateRef, watch } from 'vue'
import { watchDebounced } from '@vueuse/core'
import { useBoardFilters } from '@/composables/useBoardFilters'
import { useKeyboardShortcuts } from '@/composables/useKeyboardShortcuts'

const props = defineProps<{
  /** Every label used on the board. */
  labels: string[]
}>()

const { searchText, activeLabels, isFiltering, setSearchText, toggleLabel, clearFilters } =
  useBoardFilters()

// The input needs its own state: it changes on every keystroke, the URL only after a pause.
const draft = ref(searchText.value)

// Input → URL, once typing has stopped for 300ms.
watchDebounced(draft, (text) => setSearchText(text), { debounce: 300 })

// URL → input, for changes that didn't come from typing: back/forward, or "Clear filters".
watch(searchText, (text) => {
  if (text !== draft.value.trim()) draft.value = text
})

const searchInput = useTemplateRef('searchInput')
useKeyboardShortcuts({ '/': () => searchInput.value?.focus() })

// Also show labels that are in the URL but no longer on the board, so they can be switched off.
const allLabels = computed(() => [...new Set([...props.labels, ...activeLabels.value])])
</script>

<template>
  <section class="filters" aria-label="Filter cards">
    <div class="filters__row">
      <input
        ref="searchInput"
        v-model="draft"
        class="filters__search"
        type="search"
        placeholder="Search cards (press /)"
        aria-label="Search cards"
        aria-keyshortcuts="/"
      />

      <ul v-if="allLabels.length > 0" class="filters__labels" aria-label="Filter by label">
        <li v-for="label in allLabels" :key="label">
          <button
            class="filters__label"
            :class="{ 'filters__label--active': activeLabels.includes(label) }"
            type="button"
            :aria-pressed="activeLabels.includes(label)"
            @click="toggleLabel(label)"
          >
            {{ label }}
          </button>
        </li>
      </ul>

      <button v-if="isFiltering" class="filters__clear" type="button" @click="clearFilters">
        Clear filters
      </button>
    </div>

    <p v-if="isFiltering" class="filters__note" role="status">
      Drag and drop is paused while filters are on: with some cards hidden, a drop position wouldn't
      map to a clear place in the full list.
    </p>
  </section>
</template>

<style scoped>
.filters {
  margin-bottom: var(--space-3);
}

.filters__row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-2);
}

.filters__search {
  width: 14rem;
  padding: var(--space-1) var(--space-2);
  font: inherit;
  font-size: 0.875rem;
  color: inherit;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
}

.filters__search:focus {
  border-color: var(--color-accent);
  outline: 2px solid var(--color-label-bg);
}

.filters__labels {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-1);
  margin: 0;
  padding: 0;
  list-style: none;
}

.filters__label {
  padding: 0 var(--space-2);
  font: inherit;
  font-size: 0.8125rem;
  line-height: 1.6;
  color: var(--color-text-muted);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 999px;
  cursor: pointer;
}

.filters__label:hover {
  border-color: var(--color-accent);
}

.filters__label--active {
  color: var(--color-label-text);
  background: var(--color-label-bg);
  border-color: var(--color-accent);
}

.filters__clear {
  padding: 0 var(--space-2);
  font: inherit;
  font-size: 0.8125rem;
  color: var(--color-accent);
  background: none;
  border: none;
  cursor: pointer;
}

.filters__clear:hover {
  text-decoration: underline;
}

.filters__note {
  margin: var(--space-2) 0 0;
  font-size: 0.8125rem;
  color: var(--color-text-muted);
}
</style>
