export type MapEntry = {
  author?: string
  date?: string
  description?: string
  imageUrl?: string
  gifUrl?: string
  imageUrls?: string[]
  modes?: string[]
  players?: string
  // Preview URLs populated by the server from asset roots.
  // Frontend should handle 404s gracefully — not all maps have previews.
  stillUrl?: string
  webpUrl?: string
  svoxUrl?: string
}

export type Catalog = {
  maps: Record<string, MapEntry>
}

export type BrowseMapEntry = MapEntry & {
  name: string
}
