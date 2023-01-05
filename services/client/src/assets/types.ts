import type { DownloadState } from '../types'

export enum ResponseType {
  State,
  Data,
  Index,
}

// A file inside of an asset bundle.
export type BundleEntry = {
  filename: string
  start: number
  end: number
  audio?: number
}

export type BundleData = {
  directories: string[][]
  files: BundleEntry[]
  size: number
  dataOffset: number
  buffer: ArrayBuffer
}

export type AssetData = {
  path: string
  data: Uint8Array
}

export type MountData = {
  files: AssetData[]
  buffers: ArrayBuffer[]
}

export type Asset = {
  id: string
  path: string
}

export type IndexAsset = [
  // The index of an asset in AssetSource.assets
  number,
  // The path at which this asset should be mounted
  string
]

export type GameMap = {
  id: string
  name: string
  ogz: number
  bundle: Maybe<string>
  assets: IndexAsset[]
  image: Maybe<string>
  description: string
}

export type GameMod = {
  id: string
  name: string
  image: Maybe<string>
  description: string
}

export type Bundle = {
  id: string
  desktop: boolean
  web: boolean
  assets: IndexAsset[]
}

export type Model = {
  id: string
  name: string
}

export type AssetSource = {
  source: string
  assets: string[]
  textures: IndexAsset[]
  bundles: Bundle[]
  maps: GameMap[]
  models: Model[]
  mods: GameMod[]
}

export type AssetIndex = AssetSource[]

export enum AssetLoadStateType {
  Waiting,
  Downloading,
  Ok,
  Failed,
}

export type AssetWaitingState = {
  type: AssetLoadStateType.Waiting
}

export type AssetDownloadingState = {
  type: AssetLoadStateType.Downloading
} & DownloadState

export type AssetOkState = {
  type: AssetLoadStateType.Ok
}

export type AssetFailedState = {
  type: AssetLoadStateType.Failed
}

export type AssetLoadState =
  | AssetWaitingState
  | AssetDownloadingState
  | AssetOkState
  | AssetFailedState

export enum AssetLoadType {
  Asset,
  Bundle,
}

export type AssetState = {
  type: AssetLoadType
  // The id of the asset or bundle
  id: string
  state: AssetLoadState
}

export type AssetStateResponse = {
  op: ResponseType.State
  // The id provided in the original AssetLoadRequest
  id: string
  status: AssetLoadStateType
  state: AssetState[]
}

export type AssetDataResponse = {
  op: ResponseType.Data
  // The id provided in the original AssetLoadRequest
  id: string
  data: AssetData[]
}

export type IndexResponse = {
  op: ResponseType.Index
  index: AssetIndex
}

export type AssetResponse =
  | AssetStateResponse
  | AssetDataResponse
  | IndexResponse

export enum RequestType {
  Environment,
  Load,
}

export type AssetEnvironmentRequest = {
  op: RequestType.Environment
  assetSources: string[]
}

export enum LoadRequestType {
  Map,
  Model,
  Texture,
  Mod,
}

export type AssetLoadRequest = {
  op: RequestType.Load
  id: string
  type: LoadRequestType
  // The identifier for the data in question:
  // - Map: the map's ID or its reserved name (e.g. "dust2")
  // - Model: the model's ID or its path (e.g. "skull/blue")
  // - Texture: the texture's ID or its full path (e.g. "packages/unnamed/test.png")
  // - Mod: the mod's ID or its name (e.g. "base")
  target: string
}

export type AssetRequest = AssetEnvironmentRequest | AssetLoadRequest
