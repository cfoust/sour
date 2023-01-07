import type { GameMod } from './types'

const DEFAULT_IMAGE = 'data/logo_1024.png'
export function getModImage(mod: GameMod): string {
  const { id, image } = mod
  if (image == null) return DEFAULT_IMAGE
  return `packages/textures/images/${image}`
}
