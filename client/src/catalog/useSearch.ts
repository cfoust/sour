import * as React from 'react'

import type { BrowseMapEntry } from './types'

export type SortField = 'name' | 'date'

export function useSearch(maps: BrowseMapEntry[]) {
  const [query, setQuery] = React.useState('')
  const [sortBy, setSortBy] = React.useState<SortField>('name')

  const results = React.useMemo(() => {
    const q = query.toLowerCase()

    let filtered = maps
    if (q) {
      filtered = maps.filter((m) => {
        return (
          m.name.toLowerCase().includes(q) ||
          (m.author && m.author.toLowerCase().includes(q)) ||
          (m.description && m.description.toLowerCase().includes(q))
        )
      })
    }

    filtered = [...filtered]
    filtered.sort((a, b) => {
      if (sortBy === 'date') {
        const da = a.date || ''
        const db = b.date || ''
        return db.localeCompare(da)
      }
      return a.name.localeCompare(b.name)
    })

    return filtered
  }, [maps, query, sortBy])

  return { query, setQuery, sortBy, setSortBy, results }
}
