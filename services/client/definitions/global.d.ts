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
  postRun: Array<(() => void)>
}
declare const Module: ModuleType

declare const ASSET_PREFIX: string

type BananaBreadType = {
  execute: (command: string) => void
  renderprogress: (progress: number, text: string) => void
}
declare const BananaBread: BananaBreadType
