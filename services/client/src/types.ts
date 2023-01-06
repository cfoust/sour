export enum GameStateType {
  PageLoading,
  // Waiting for files to download
  Downloading,
  // When we're starting a map transition
  MapChange,
  // The game is starting up
  Running,
  // At the main menu
  Ready,
  GameError,
}

export type PageLoadingState = {
  type: GameStateType.PageLoading
}

export enum DownloadingType {
  Map,
  Mod,
  Index,
}

export type DownloadState = {
  downloadedBytes: number
  totalBytes: number
}

export type DownloadingState = {
  type: GameStateType.Downloading
  downloadType: DownloadingType
} & DownloadState

export type MapChangeState = {
  type: GameStateType.MapChange
  map: string
}

export type RunningState = {
  type: GameStateType.Running
}

export type ReadyState = {
  type: GameStateType.Ready
}

export type ErrorState = {
  type: GameStateType.GameError
}

export type GameState =
  | PageLoadingState
  | DownloadingState
  | MapChangeState
  | RunningState
  | ReadyState
  | ErrorState
