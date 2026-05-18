import * as React from 'react'
import type { BrowseMapEntry } from '../catalog/types'

export type Filters = {
  year: number
  hasScreenshot: boolean
}

export function useFilters(maps: BrowseMapEntry[]) {
  const [filters, setFilters] = React.useState<Filters>({
    year: 2020,
    hasScreenshot: false,
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

    return result
  }, [maps, filters, yearRange])

  return {
    filters,
    filtered,
    yearRange,
    mapsWithScreenshots,
    setYear: (year: number) => setFilters(f => ({ ...f, year })),
    toggleHasScreenshot: () => setFilters(f => ({ ...f, hasScreenshot: !f.hasScreenshot })),
  }
}
