import type { DBSchema, IDBPDatabase } from 'idb'
import { openDB } from 'idb'

interface BundleDB extends DBSchema {
  bundles: {
    key: string
    value: ArrayBuffer
  }
}

async function initDB(): Promise<IDBPDatabase<BundleDB>> {
  return await openDB<BundleDB>('sour-assets', 1, {
    upgrade(db) {
      db.createObjectStore('bundles')
    },
  })
}

export async function haveBundle(
  target: string
): Promise<boolean> {
  const db = await initDB()
  const keys = await db.getAllKeys('bundles')
  return keys.includes(target)
}

export async function getBundle(
  target: string
): Promise<Maybe<ArrayBuffer>> {
  const db = await initDB()
  const bundle = await db.get('bundles', target)
  if (bundle == null) return null
  return bundle
}

export async function saveAsset(
  target: string,
  buffer: ArrayBuffer
) {
  const db = await initDB()
  await db.put('bundles', buffer, target)
}
