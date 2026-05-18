import * as React from 'react'
import { Box, Image, Text, Button } from '@chakra-ui/react'

import type { BrowseMapEntry } from './types'

const PLACEHOLDER =
  'data:image/svg+xml,' +
  encodeURIComponent(
    '<svg xmlns="http://www.w3.org/2000/svg" width="256" height="192" fill="%231a202c"><rect width="256" height="192"/><text x="128" y="96" text-anchor="middle" dy=".3em" fill="%23718096" font-size="14">No image</text></svg>'
  )

type MapCardProps = {
  entry: BrowseMapEntry
  onPlay: () => void
}

export default function MapCard({ entry, onPlay }: MapCardProps) {
  const [imgSrc, setImgSrc] = React.useState(entry.imageUrl || PLACEHOLDER)

  return (
    <Box
      borderWidth="1px"
      borderRadius="md"
      overflow="hidden"
      bg="gray.800"
      opacity={entry.available ? 1 : 0.5}
    >
      <Image
        src={imgSrc}
        alt={entry.name}
        width="100%"
        height="180px"
        objectFit="cover"
        onError={() => setImgSrc(PLACEHOLDER)}
      />
      <Box p={3}>
        <Text fontWeight="bold" fontSize="md" isTruncated>
          {entry.name}
        </Text>
        {(entry.author || entry.date) && (
          <Text fontSize="xs" color="gray.400" isTruncated>
            {entry.author}
            {entry.author && entry.date && ' · '}
            {entry.date}
          </Text>
        )}
        {entry.description && (
          <Text fontSize="sm" color="gray.300" noOfLines={2} mt={1}>
            {entry.description}
          </Text>
        )}
        <Button
          size="sm"
          colorScheme="blue"
          mt={2}
          width="100%"
          onClick={onPlay}
          isDisabled={!entry.available}
        >
          Play
        </Button>
      </Box>
    </Box>
  )
}
