import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import type { LocationQueryValue } from 'vue-router'
import type { Card } from '@/types'

type QueryParam = LocationQueryValue | LocationQueryValue[] | undefined

/** `?label=a&label=b` arrives as an array, `?label=a` as a string, and a bare `?label` as null. */
function toStrings(param: QueryParam): string[] {
  const values = Array.isArray(param) ? param : [param]
  return values.filter((value): value is string => typeof value === 'string' && value !== '')
}

/**
 * Board filters, with the URL query as the single source of truth:
 * `?q=text` for search and `?label=bug&label=ui` for labels.
 *
 * Nothing is stored here. Every value is derived from the route with `computed`, so any
 * component that calls this sees the same filters, and they survive a refresh.
 */
export function useBoardFilters() {
  const route = useRoute()
  const router = useRouter()

  const searchText = computed(() => toStrings(route.query.q)[0]?.trim() ?? '')
  const activeLabels = computed(() => toStrings(route.query.label))
  const isFiltering = computed(() => searchText.value !== '' || activeLabels.value.length > 0)

  /** A predicate for the current filters: text in title or description, and any active label. */
  const matches = computed(() => {
    const text = searchText.value.toLowerCase()
    const labels = activeLabels.value

    return (card: Card): boolean => {
      const matchesText =
        text === '' ||
        card.title.toLowerCase().includes(text) ||
        card.description.toLowerCase().includes(text)
      const matchesLabel =
        labels.length === 0 || labels.some((label) => card.labels.includes(label))
      return matchesText && matchesLabel
    }
  })

  // `replace`, not `push`: filter changes shouldn't each add a history entry.
  // A key set to undefined is dropped from the URL.
  function setSearchText(text: string) {
    const q = text.trim()
    if (q === searchText.value) return
    router.replace({ query: { ...route.query, q: q || undefined } })
  }

  function toggleLabel(label: string) {
    const labels = activeLabels.value.includes(label)
      ? activeLabels.value.filter((active) => active !== label)
      : [...activeLabels.value, label]
    router.replace({ query: { ...route.query, label: labels.length > 0 ? labels : undefined } })
  }

  function clearFilters() {
    router.replace({ query: { ...route.query, q: undefined, label: undefined } })
  }

  return {
    searchText,
    activeLabels,
    isFiltering,
    matches,
    setSearchText,
    toggleLabel,
    clearFilters,
  }
}
