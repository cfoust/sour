export type MapEntry = {
  author?: string
  date?: string
  description?: string
  imageUrl?: string
  gifUrl?: string
}

export type Catalog = {
  maps: Record<string, MapEntry>
}

export type BrowseMapEntry = MapEntry & {
  name: string
  available: boolean
}
