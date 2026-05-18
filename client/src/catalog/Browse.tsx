import * as React from 'react'
import {
  Box,
  Flex,
  Heading,
  Input,
  Select,
  SimpleGrid,
  Button,
  Text,
  Center,
  Spinner,
} from '@chakra-ui/react'

import type { BrowseMapEntry } from './types'
import { useSearch } from './useSearch'
import MapCard from './MapCard'

const PAGE_SIZE = 50

type BrowseProps = {
  maps: BrowseMapEntry[]
  loading: boolean
  onPlay: (mapName: string) => void
}

export default function Browse({ maps, loading, onPlay }: BrowseProps) {
  const { query, setQuery, sortBy, setSortBy, results } = useSearch(maps)
  const [shown, setShown] = React.useState(PAGE_SIZE)

  React.useEffect(() => {
    setShown(PAGE_SIZE)
  }, [query, sortBy])

  if (loading) {
    return (
      <Center width="100%" height="100%">
        <Spinner size="xl" color="blue.300" />
      </Center>
    )
  }

  const visible = results.slice(0, shown)
  const hasMore = shown < results.length

  return (
    <Box
      width="100%"
      height="100%"
      overflow="auto"
      bg="gray.900"
      color="white"
      p={6}
    >
      <Flex direction="column" maxW="1200px" mx="auto">
        <Heading size="lg" mb={4}>
          Maps
        </Heading>

        <Flex mb={4} gap={3}>
          <Input
            placeholder="Search maps..."
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            bg="gray.800"
            flex={1}
          />
          <Select
            value={sortBy}
            onChange={(e) => setSortBy(e.target.value as 'name' | 'date')}
            bg="gray.800"
            width="150px"
          >
            <option value="name">Name</option>
            <option value="date">Date</option>
          </Select>
        </Flex>

        <Text fontSize="sm" color="gray.400" mb={4}>
          {results.length} map{results.length !== 1 ? 's' : ''}
          {query && ` matching "${query}"`}
        </Text>

        <SimpleGrid columns={[1, 2, 3, 4]} spacing={4}>
          {visible.map((entry) => (
            <MapCard
              key={entry.name}
              entry={entry}
              onPlay={() => onPlay(entry.name)}
            />
          ))}
        </SimpleGrid>

        {hasMore && (
          <Center mt={6} mb={4}>
            <Button
              onClick={() => setShown((s) => s + PAGE_SIZE)}
              colorScheme="blue"
              variant="outline"
            >
              Load more ({results.length - shown} remaining)
            </Button>
          </Center>
        )}
      </Flex>
    </Box>
  )
}
