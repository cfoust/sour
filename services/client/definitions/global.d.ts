type PreloadFile = {
  filename: string
  start: number
  end: number
  audio: number
}
type PreloadNode = {
  name: string
  pointer: number
  files: PreloadFile[]
}

// Lets us use the Module API in a type safe way
type ModuleType = {
  tweakDetail: () => void
  postLoadWorld: () => void
  desiredWidth: number
  desiredHeight: number
  canvas: HTMLCanvasElement | null
  setCanvasSize: ((width: number, height: number) => void) | null
  setStatus: (text: string) => void
  print: (text: string) => void
  removeRunDependency: (file: string) => void
  monitorRunDependencies: (left: number) => void
  registerNode: (node: PreloadNode) => void
  preInit: Array<() => void>
  postRun: Array<() => void>
  run: () => void
  _free: (pointer: number) => void
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
  renderprogress: (progress: number, text: string) => void
}
declare const BananaBread: BananaBreadType
declare let shouldRunNow: boolean
