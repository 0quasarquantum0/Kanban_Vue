import { computed, inject, provide, watchEffect } from 'vue'
import type { ComputedRef, InjectionKey } from 'vue'
import { usePreferredDark, useStorage } from '@vueuse/core'

export type Theme = 'light' | 'dark'

export interface ThemeContext {
  theme: ComputedRef<Theme>
  toggleTheme: () => void
}

// The key carries the value's type, so `inject(themeKey)` is typed without a cast.
// A Symbol can't collide with a key provided by another library.
const themeKey: InjectionKey<ThemeContext> = Symbol('theme')

/** Call once in a root component. Every descendant can then call `useTheme()`. */
export function provideTheme(): ThemeContext {
  const prefersDark = usePreferredDark()
  // 'auto' follows the operating system until the user picks a theme.
  const saved = useStorage<Theme | 'auto'>('kanban-theme', 'auto')

  const theme = computed<Theme>(() => {
    if (saved.value === 'light' || saved.value === 'dark') return saved.value
    return prefersDark.value ? 'dark' : 'light'
  })

  // The CSS in main.css switches its variables on this attribute.
  watchEffect(() => {
    document.documentElement.dataset.theme = theme.value
  })

  function toggleTheme() {
    saved.value = theme.value === 'dark' ? 'light' : 'dark'
  }

  const context: ThemeContext = { theme, toggleTheme }
  provide(themeKey, context)
  return context
}

export function useTheme(): ThemeContext {
  const context = inject(themeKey)
  if (!context) throw new Error('useTheme() needs provideTheme() to be called in an ancestor')
  return context
}
