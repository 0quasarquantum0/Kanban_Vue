<script setup lang="ts">
import { onBeforeUnmount, onMounted } from 'vue'
import { RouterView } from 'vue-router'
import SyncNotice from '@/components/SyncNotice.vue'
import { provideTheme } from '@/composables/useTheme'
import { useKanbanStore } from '@/stores/kanban'

// Makes the theme available to every component below, however deep.
provideTheme()

// The data lives on the server now, so nothing can render until it has arrived.
const store = useKanbanStore()
store.ensureLoaded()

// Live updates: when another tab changes something, the server says so and the store reloads.
onMounted(() => store.startListening())
onBeforeUnmount(() => store.stopListening())

// A full reload re-runs the route guards against the freshly loaded data.
function retry() {
  window.location.reload()
}
</script>

<template>
  <main v-if="store.status === 'error'" class="sync-screen" role="alert">
    <h1>Can't load your boards</h1>
    <p>{{ store.loadError }}</p>
    <p class="sync-screen__hint">Is the Go server running? Start it from <code>server/</code>.</p>
    <button class="button" type="button" @click="retry">Try again</button>
  </main>

  <main v-else-if="store.status !== 'ready'" class="sync-screen" role="status">
    <p>Loading your boards…</p>
  </main>

  <template v-else>
    <RouterView />
    <SyncNotice v-if="store.notice" :message="store.notice" @dismiss="store.dismissNotice()" />
  </template>
</template>

<style scoped>
.sync-screen {
  display: grid;
  justify-items: center;
  gap: var(--space-2);
  padding: var(--space-6);
  text-align: center;
}

.sync-screen__hint {
  color: var(--color-text-muted);
}
</style>
