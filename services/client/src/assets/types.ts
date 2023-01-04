import type { DownloadState } from '../types'

export enum ResponseType {
  State,
  Bundle,
  Index,
}

// A file inside of an asset bundle.
export type BundleEntry = {
  filename: string
  start: number
  end: number
  audio?: number
}

export type Bundle = {
  directories: string[][]
  files: BundleEntry[]
  size: number
  dataOffset: number
  buffer: ArrayBuffer
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

export type AssetData = {
  path: string
  data: ArrayBuffer
}

export type GameMap = {
  id: string
  name: string
  ogz: number
  assets: IndexAsset[]
  image: Maybe<string>
  description: string
}

export type GameMod = {
  name: string
  assets: IndexAsset[]
}

export type AssetSource = {
  source: string
  assets: string[]
  maps: GameMap[]
  mods: GameMod[]
}

export type BundleIndex = AssetSource[]

export enum BundleLoadStateType {
  Waiting,
  Downloading,
  Ok,
  Failed,
}

export type BundleWaitingState = {
  type: BundleLoadStateType.Waiting
}

export type BundleDownloadingState = {
  type: BundleLoadStateType.Downloading
} & DownloadState

export type BundleOkState = {
  type: BundleLoadStateType.Ok
}

export type BundleFailedState = {
  type: BundleLoadStateType.Failed
}

export type BundleLoadState =
  | BundleWaitingState
  | BundleDownloadingState
  | BundleOkState
  | BundleFailedState

export type BundleState = {
  name: string
  state: BundleLoadState
}

export type AssetStateResponse = {
  op: ResponseType.State
  state: BundleState[]
}

export type AssetBundleResponse = {
  op: ResponseType.Bundle
  target: string
  id: string
  data: AssetData[]
}

export type IndexResponse = {
  op: ResponseType.Index
  index: BundleIndex
}

export type AssetResponse =
  | AssetStateResponse
  | AssetBundleResponse
  | IndexResponse

export enum RequestType {
  Environment,
  Load,
}

export type AssetEnvironmentRequest = {
  op: RequestType.Environment
  assetSources: string[]
}

export type AssetLoadRequest = {
  op: RequestType.Load
  id: string
  target: string
}

export type AssetRequest = AssetEnvironmentRequest | AssetLoadRequest
