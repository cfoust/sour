// Lets us use the Module API in a type safe way
type ModuleType = {
  desiredWidth: number
  desiredHeight: number
  canvas: HTMLCanvasElement | null
  setCanvasSize: ((width: number, height: number) => void) | null
  setStatus: (text: string) => void
}
declare const Module: ModuleType
