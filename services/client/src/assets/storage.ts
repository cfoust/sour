import type { DBSchema, IDBPDatabase } from 'idb'
import { openDB } from 'idb'

interface BundleDB extends DBSchema {
  blobs: {
    key: string
    value: ArrayBuffer
  }
}

async function initDB(): Promise<IDBPDatabase<BundleDB>> {
  return await openDB<BundleDB>('sour-assets', 1, {
    upgrade(db) {
      db.createObjectStore('blobs')
    },
  })
}

export async function haveBlob(
  target: string
): Promise<boolean> {
  const db = await initDB()
  const keys = await db.getAllKeys('blobs')
  return keys.includes(target)
}

export async function getBlob(
  target: string
): Promise<Maybe<ArrayBuffer>> {
  const db = await initDB()
  const bundle = await db.get('blobs', target)
  if (bundle == null) return null
  return bundle
}

export async function saveBlob(
  target: string,
  buffer: ArrayBuffer
) {
  const db = await initDB()
  await db.put('blobs', buffer, target)
}
