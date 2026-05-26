import * as React from 'react'

// Map name → asset source base URL
let mapSourceLookup: Record<string, string> = {}
let version = 0
const listeners = new Set<() => void>()

export function registerMapSources(sources: Record<string, string>) {
  mapSourceLookup = sources
  version++
  listeners.forEach(fn => fn())
}

function useMapSources(): number {
  const [v, setV] = React.useState(version)
  React.useEffect(() => {
    const cb = () => setV(++version)
    listeners.add(cb)
    // If sources already loaded, trigger immediately
    if (Object.keys(mapSourceLookup).length > 0) setV(version)
    return () => { listeners.delete(cb) }
  }, [])
  return v
}

export function usePreviewStill(mapName: string): string | null {
  useMapSources()
  const base = mapSourceLookup[mapName]
  if (!base) return null
  return `${base}previews/${mapName}.jpg`
}

export function usePreviewSvox(mapName: string): string | null {
  useMapSources()
  const base = mapSourceLookup[mapName]
  if (!base) return null
  return `${base}previews/${mapName}.svox`
}
