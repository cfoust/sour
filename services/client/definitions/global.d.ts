// Lets us use the Module API in a type safe way
type ModuleType = {
  desiredWidth: number
  desiredHeight: number
  canvas: HTMLCanvasElement | null
}
declare const Module: ModuleType
