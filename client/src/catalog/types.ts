export type MapEntry = {
  author?: string
  date?: string
  description?: string
  imageUrl?: string
  gifUrl?: string
  imageUrls?: string[]
  modes?: string[]
  players?: string
}

export type Catalog = {
  maps: Record<string, MapEntry>
}

export type BrowseMapEntry = MapEntry & {
  name: string
}
