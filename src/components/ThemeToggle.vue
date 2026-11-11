<script setup lang="ts">
import { useTheme } from '@/composables/useTheme'

// Injected from App.vue. Nothing between App and this component passes the theme along.
const { theme, toggleTheme } = useTheme()
</script>

<template>
  <button class="button" type="button" @click="toggleTheme">
    {{ theme === 'dark' ? 'Light theme' : 'Dark theme' }}
  </button>
</template>
