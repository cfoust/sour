import * as React from 'react'

export default function FileDropper(props: { children: JSX.Element }) {
  const { children } = props
  const [hovered, setHovered] = React.useState<boolean>(false)

  const onDrop = React.useCallback((event: DragEvent) => {
    event.preventDefault()
    const { dataTransfer } = event
    if (dataTransfer == null) return
    const { files } = dataTransfer
    if (files == null) return
    event.stopPropagation()
    event.preventDefault()
    console.log(files);
    setHovered(false)
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
