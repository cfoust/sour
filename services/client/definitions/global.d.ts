type PreloadFile = {
  filename: string
  start: number
  end: number
  audio?: number
}
type PreloadNode = {
  name: string
  files: PreloadFile[]
}

// Lets us use the Module API in a type safe way
type ModuleType = {
  HEAPU8: Uint8Array
  _free: (pointer: number) => void
  _malloc: (length: number) => number
  canvas: HTMLCanvasElement | null
  desiredHeight: number
  desiredWidth: number
  monitorRunDependencies: (left: number) => void
  postLoadWorld: () => void
  postRun: Array<() => void>
  preRun: Array<() => void>
  preInit: Array<() => void>
  onGameReady: () => void
  print: (text: string) => void
  registerNode: (node: PreloadNode) => void
  removeRunDependency: (file: string) => void
  run: () => void
  setCanvasSize: ((width: number, height: number) => void) | null
  setStatus: (text: string) => void
  tweakDetail: () => void

  calledRun: boolean
  FS_createPath: (...path: Array<string, boolean>) => void
  FS_createPreloadedFile: (parent: string, name: Maybe<string>, url: string | Uint8Array, canRead: boolean, canWrite: boolean, onload: () => void, onerror: () => void, dontCreateFile: boolean, canOwn: boolean, preFinish?: () => void) => void
  addRunDependency: (dependency: string) => void
}
declare const Module: ModuleType
declare type Maybe<T> = T | null | undefined

declare const FS: {
  unlink: (file: string) => void
}

declare const ASSET_PREFIX: string
declare const GAME_SERVER: string

type BananaBreadType = {
  execute: (command: string) => void
  loadWorld: (map: string, cmap?: string) => void
  renderprogress: (progress: number, text: string) => void
  injectServer: (
    host: string,
    port: number,
    infoPointer: number,
    infoLength: number
  ) => void
}
declare const BananaBread: BananaBreadType
declare let shouldRunNow: boolean
declare let calledRun: boolean
