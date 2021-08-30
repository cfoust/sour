export type ClientID = number

export type User = {
  id: ClientID
  // The visible username of the user
  name: string
  // The location of the user in the game world
  position: Vec3
  // Whether this user is speaking.
  speaking: boolean
  // Whether this user is muted.
  muted: boolean
}

// Contains the position of entities
export type EntityState = {
  // The current user
  me: User,

  // Everyone else
  users: User[],
}

export enum GameStateType {
  PageLoading,
  // Waiting for files to download
  Downloading,
  // When we're starting a map transition
  MapChange,
  // The game is starting up
  Running,
  // Attempting to connect to server
  Connecting,
  Connected,
  GameError,
}

export type PageLoadingState = {
  type: GameStateType.PageLoading
}

export type DownloadingState = {
  type: GameStateType.Downloading
  downloadedBytes: number
  totalBytes: number
}

export type MapChangeState = {
  type: GameStateType.MapChange
  map: string
}

export type RunningState = {
  type: GameStateType.Running
}

export type ConnectingState = {
  type: GameStateType.Connecting
}

export type ConnectedState = {
  type: GameStateType.Connected
}

export type ErrorState = {
  type: GameStateType.GameError
}

export type GameState =
  | PageLoadingState
  | DownloadingState
  | MapChangeState
  | RunningState
  | ConnectingState
  | ConnectedState
  | ErrorState
