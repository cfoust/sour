import * as React from 'react'

import { mountFile } from './assets/hook'

export async function handleUpload(file: File) {
  const { name } = file

  if (name.endsWith('.dmo')) {
    const path = `demo/${name}`
    const buffer = await file.arrayBuffer()
    await mountFile(path, new Uint8Array(buffer))
    BananaBread.execute(`demo ${name}`)
  }

  if (name.endsWith('.ogz')) {
    const path = `packages/base/${name}`
    const mapName = name.slice(0, -4)
    const buffer = await file.arrayBuffer()
    await mountFile(path, new Uint8Array(buffer))
    BananaBread.execute(`map ${mapName}`)
  }
}

export default function FileDropper(props: { children: JSX.Element }) {
  const { children } = props
  const [hovered, setHovered] = React.useState<boolean>(false)

  const onDrop = React.useCallback((event: DragEvent) => {
    event.preventDefault()
    const { dataTransfer } = event
    if (dataTransfer == null) return
    const { files } = dataTransfer
    if (files == null || files.length !== 1) return
    event.stopPropagation()
    event.preventDefault()
    setHovered(false)
    handleUpload(files[0])
  }, [])

  const onDragOver = React.useCallback((event: DragEvent) => {
    const { dataTransfer } = event
    if (dataTransfer == null) return
    const { types } = dataTransfer
    if (types.length !== 1) return
    const [type_] = types
    if (type_ !== 'Files') return
    event.stopPropagation()
    event.preventDefault()
    setHovered(true)
    dataTransfer.dropEffect = 'copy'
  }, [])

  const onDragLeave = React.useCallback((event: DragEvent) => {
    event.preventDefault()
    setHovered(false)
  }, [])

  React.useEffect(() => {
    document.addEventListener('dragover', onDragOver)
    document.addEventListener('drop', onDrop)
    document.addEventListener('dragleave', onDragLeave)
    return () => {
      document.removeEventListener('dragover', onDragOver)
      document.removeEventListener('drop', onDrop)
      document.removeEventListener('dragleave', onDragLeave)
    }
  }, [onDrop, onDragOver, onDragLeave])

  return <>{hovered && children}</>
}
