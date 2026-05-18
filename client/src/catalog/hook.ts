import * as React from 'react'

import type { Catalog } from './types'
import { CONFIG } from '../config'

export function useCatalog(): {
  catalog: Maybe<Catalog>
  loading: boolean
} {
  const [catalog, setCatalog] = React.useState<Maybe<Catalog>>(null)
  const [loading, setLoading] = React.useState(false)

  React.useEffect(() => {
    if (!CONFIG.catalog) return

    setLoading(true)
    fetch(CONFIG.catalog)
      .then((res) => res.json())
      .then((data: Catalog) => {
        setCatalog(data)
        setLoading(false)
      })
      .catch((err) => {
        console.error('Failed to load catalog:', err)
        setLoading(false)
      })
  }, [])

  return { catalog, loading }
}
