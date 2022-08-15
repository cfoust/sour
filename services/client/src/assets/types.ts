import type { DownloadState } from '../types'

export enum ResponseType {
  State,
  Bundle,
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
  bundle: Bundle
}

export type AssetResponse = AssetStateResponse | AssetBundleResponse

export enum RequestType {
  Environment,
  Load,
}

export type AssetEnvironmentRequest = {
  op: RequestType.Environment
  ASSET_PREFIX: string
}

export type AssetLoadRequest = {
  op: RequestType.Load
  id: symbol
  target: string
}

export type AssetRequest = AssetEnvironmentRequest | AssetLoadRequest
