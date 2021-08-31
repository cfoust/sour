declare type Vec2 = [number, number];
declare type Vec3 = [number, number, number];
declare type Vec4 = [number, number, number, number];
declare type Quat = [number, number, number, number];

declare type Maybe<T> = T | null | undefined

// Lets us use the Module API in a type safe way
type ModuleType = {
  canvas: HTMLCanvasElement | null
  desiredHeight: number
  desiredWidth: number
  postLoadWorld: () => void
  postRun: Array<(() => void)>
  onPlayerMove: (client: number, clientPos: Vec3, ourPos: Vec3) => void
  onPlayerJoin: (client: number) => void
  // When we get our client number assigned
  onClientNumber: (client: number) => void
  onPlayerName: (client: number, name: string) => void
  onPlayerLeave: (client: number) => void
  print: (text: string) => void
  removeRunDependency: (file: string) => void
  setCanvasSize: ((width: number, height: number) => void) | null
  setStatus: (text: string) => void
  tweakDetail: () => void
}
declare const Module: ModuleType

declare const ASSET_PREFIX: string

type BananaBreadType = {
  execute: (command: string) => void
}
declare const BananaBread: BananaBreadType

declare type IntervalID = ReturnType<typeof setInterval>
