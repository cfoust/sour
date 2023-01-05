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

export enum LoadStateType {
  // The request is in-flight
  Waiting,
  // No assets of this type were found
  Missing,
  Downloading,
  Ok,
  // There was an operational fault while responding to this request
  Failed,
}

export type WaitingState = {
  type: LoadStateType.Waiting
}

export type MissingState = {
  type: LoadStateType.Missing
}

export type DownloadingState = {
  type: LoadStateType.Downloading
} & DownloadState

export type OkState = {
  type: LoadStateType.Ok
  totalBytes: number
}

export type FailedState = {
  type: LoadStateType.Failed
}

export const load = {
  waiting: (): WaitingState => ({
    type: LoadStateType.Waiting,
  }),
  missing: (): MissingState => ({
    type: LoadStateType.Missing,
  }),
  downloading: (state: DownloadState): DownloadingState => ({
    type: LoadStateType.Downloading,
    ...state,
  }),
  ok: (totalBytes: number): OkState => ({
    type: LoadStateType.Ok,
    totalBytes,
  }),
  failed: (): FailedState => ({
    type: LoadStateType.Failed,
  }),
}

export type LoadState =
  | WaitingState
  | MissingState
  | DownloadingState
  | OkState
  | FailedState

export enum DataType {
  Asset,
  Bundle,
}

export type AssetState = {
  // The id of the asset or bundle
  id: string
  type: DataType
  state: LoadState
}

export type StateResponse = {
  op: ResponseType.State
  // The id provided in the original AssetLoadRequest
  id: string
  // The high-level status, which generally represents the aggregation
  // of all of the assets in `state`
  overall: LoadState
  // The state for specific assets or bundles
  individual: AssetState[]
}

export type DataResponse = {
  op: ResponseType.Data
  // The id provided in the original AssetLoadRequest
  id: string
  data: AssetData[]
}

export type IndexResponse = {
  op: ResponseType.Index
  index: AssetIndex
}

export type Response = StateResponse | DataResponse | IndexResponse

export enum RequestType {
  Environment,
  Load,
}

export type EnvironmentRequest = {
  op: RequestType.Environment
  assetSources: string[]
}

export enum LoadRequestType {
  Map,
  Model,
  Texture,
  Mod,
}

export type LoadRequest = {
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

export type Request = EnvironmentRequest | LoadRequest
