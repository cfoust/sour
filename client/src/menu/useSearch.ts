import * as React from 'react'
import type { BrowseMapEntry } from '../catalog/types'

export type SortField = 'name' | 'date'

export function useSearch(maps: BrowseMapEntry[]) {
  const [query, setQuery] = React.useState('')
  const [sortBy, setSortBy] = React.useState<SortField>('name')

  const results = React.useMemo(() => {
    const q = query.toLowerCase()

    let filtered = maps
    if (q) {
      filtered = maps.filter(m =>
        m.name.toLowerCase().includes(q) ||
        (m.author && m.author.toLowerCase().includes(q)) ||
        (m.description && m.description.toLowerCase().includes(q))
      )
    }

    const sorted = [...filtered]
    sorted.sort((a, b) => {
      if (sortBy === 'date') {
        const da = a.date || ''
        const db = b.date || ''
        return db.localeCompare(da)
      }
      return a.name.localeCompare(b.name)
    })

    return sorted
  }, [maps, query, sortBy])

  return { query, setQuery, sortBy, setSortBy, results }
}
