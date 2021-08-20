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
}
declare const Module: ModuleType

type BananaBreadType = {
  execute: (command: string) => void
}
declare const BananaBread: BananaBreadType
