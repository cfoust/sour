import * as React from 'react'
import type { BrowseMapEntry } from '../catalog/types'

export type Filters = {
  year: number
  hasScreenshot: boolean
  activeModes: Set<string>
}

export type ModeCount = {
  mode: string
  count: number
}

export function useFilters(maps: BrowseMapEntry[]) {
  const [filters, setFilters] = React.useState<Filters>({
    year: 2020,
    hasScreenshot: false,
    activeModes: new Set(),
  })

  // Compute year range from actual data
  const yearRange = React.useMemo((): [number, number] => {
    let min = 2020, max = 2005
    for (const m of maps) {
      if (!m.date) continue
      const y = parseInt(m.date.slice(0, 4), 10)
      if (isNaN(y)) continue
      if (y < min) min = y
      if (y > max) max = y
    }
    return min <= max ? [min, max] : [2005, 2020]
  }, [maps])

  const mapsWithScreenshots = React.useMemo(
    () => maps.filter(m => !!m.imageUrl).length,
    [maps]
  )

  // Compute available modes and their counts
  const modeCounts = React.useMemo((): ModeCount[] => {
    const counts = new Map<string, number>()
    for (const m of maps) {
      if (!m.modes) continue
      for (const mode of m.modes) {
        counts.set(mode, (counts.get(mode) || 0) + 1)
      }
    }
    return Array.from(counts.entries())
      .map(([mode, count]) => ({ mode, count }))
      .sort((a, b) => b.count - a.count)
  }, [maps])

  const filtered = React.useMemo(() => {
    let result = maps

    // Year filter: show maps up to the selected year
    if (filters.year < yearRange[1]) {
      result = result.filter(m => {
        if (!m.date) return true
        const y = parseInt(m.date.slice(0, 4), 10)
        if (isNaN(y)) return true
        return y <= filters.year
      })
    }

    // Screenshot filter
    if (filters.hasScreenshot) {
      result = result.filter(m => !!m.imageUrl)
    }

    // Mode filter: map must have at least one of the active modes
    if (filters.activeModes.size > 0) {
      result = result.filter(m => {
        if (!m.modes) return false
        return m.modes.some(mode => filters.activeModes.has(mode))
      })
    }

    return result
  }, [maps, filters, yearRange])

  return {
    filters,
    filtered,
    yearRange,
    mapsWithScreenshots,
    modeCounts,
    setYear: (year: number) => setFilters(f => ({ ...f, year })),
    toggleHasScreenshot: () => setFilters(f => ({ ...f, hasScreenshot: !f.hasScreenshot })),
    toggleMode: (mode: string) => setFilters(f => {
      const next = new Set(f.activeModes)
      if (next.has(mode)) {
        next.delete(mode)
      } else {
        next.add(mode)
      }
      return { ...f, activeModes: next }
    }),
  }
}
