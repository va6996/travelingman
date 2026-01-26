import { useState, useMemo } from 'react'
import { VStack, Box, Text, Heading, Divider, Tabs, TabList, TabPanels, Tab, TabPanel } from '@chakra-ui/react'
import { Itinerary, Node, Edge } from '../gen/protos/graph_pb'
import { NodeCard } from './NodeCard'
import { EdgeCard } from './EdgeCard'

interface ItineraryViewProps {
    itinerary?: Itinerary
    possibleItineraries?: Itinerary[]
}

// Helper to sort graph elements by traversal (Node -> Edge -> Node)
const sortTimeline = (itinerary: Itinerary) => {
    if (!itinerary.graph) return []

    const nodes = new Map(itinerary.graph.nodes.map(n => [n.id, n]))
    const edgesByFrom = new Map<string, Edge[]>()

    itinerary.graph.edges.forEach(e => {
        const list = edgesByFrom.get(e.fromId) || []
        list.push(e)
        // Sort edges by time if possible? Usually one edge per traversal step in linear trip
        edgesByFrom.set(e.fromId, list)
    })

    // Find start node(s): Nodes that are not 'toId' of any edge
    const toIds = new Set(itinerary.graph.edges.map(e => e.toId))
    const startNodes = itinerary.graph.nodes.filter(n => !toIds.has(n.id))

    // Fallback: if loop or whatever, just pick first node
    let currentNodes = startNodes.length > 0 ? startNodes : [itinerary.graph.nodes[0]]
    // If multiple start nodes, maybe just pick one or handle valid DAG
    // For linear itinerary, usually one start.

    const timeline: { type: 'node' | 'edge', data: Node | Edge }[] = []
    const visited = new Set<string>()

    // Simple DFS or linear traversal
    const traverse = (nodeId: string) => {
        if (visited.has(nodeId)) return
        visited.add(nodeId)

        const node = nodes.get(nodeId)
        if (node) {
            timeline.push({ type: 'node', data: node })

            const outgoing = edgesByFrom.get(nodeId)
            if (outgoing) {
                // Determine order of outgoing edges? sort by departure time
                outgoing.sort((a, b) => {
                    const getDepartureTime = (edge: Edge) => {
                        if (edge.transport?.details.case === 'flight') {
                            return edge.transport.details.value.departureTime?.toDate().getTime() || 0
                        }
                        return 0
                    }
                    return getDepartureTime(a) - getDepartureTime(b)
                })

                outgoing.forEach(e => {
                    timeline.push({ type: 'edge', data: e })
                    traverse(e.toId)
                })
            }
        }
    }

    // Start traversal from identified start node(s)
    currentNodes.forEach(n => traverse(n.id))

    return timeline
}

const SingleItineraryTimeline = ({ itinerary }: { itinerary: Itinerary }) => {
    // Map of ID -> selected option index
    const [selectedOptions, setSelectedOptions] = useState<Record<string, number>>({})
    const timeline = useMemo(() => sortTimeline(itinerary), [itinerary])

    const handleSelect = (id: string, index: number) => {
        setSelectedOptions(prev => ({ ...prev, [id]: index }))
    }

    return (
        <VStack spacing={0} align="stretch" position="relative" pb={10} pt={4}>
            {itinerary.error && (
                <Box p={4} bg="red.900" color="white" borderRadius="md" mb={4}>
                    <Text fontWeight="bold">Unable to generate itinerary:</Text>
                    <Text>{itinerary.error.message}</Text>
                </Box>
            )}

            {/* Vertical Line */}
            <Box
                position="absolute"
                left="20px"
                top="0"
                bottom="0"
                width="2px"
                bg="whiteAlpha.200"
                zIndex={0}
            />

            {timeline.map((item, idx) => {
                const isNode = item.type === 'node'
                const id = isNode
                    ? (item.data as Node).id
                    : `${(item.data as Edge).fromId}-${(item.data as Edge).toId}-${idx}`

                const selectedIdx = selectedOptions[id] || 0

                return (
                    <Box key={idx} pl={10} position="relative" mb={6}>
                        {/* Dot */}
                        <Box
                            position="absolute"
                            left="13px"
                            top="28px"
                            width="16px"
                            height="16px"
                            borderRadius="full"
                            bg={isNode ? "green.400" : "#1a202c"}
                            border={isNode ? "none" : "2px solid"}
                            borderColor={isNode ? "none" : "green.500"}
                            zIndex={1}
                            shadow={isNode ? "0 0 10px rgba(72, 187, 120, 0.6)" : "none"}
                        />

                        {isNode ? (
                            <NodeCard
                                nodeId={id}
                                stay={(item.data as Node).stay!}
                                options={(item.data as Node).stayOptions}
                                selectedOptionIndex={selectedIdx}
                                onSelectOption={(i) => handleSelect(id, i)}
                                locationName={(item.data as Node).location}
                            />
                        ) : (
                            <EdgeCard
                                transport={(item.data as Edge).transport!}
                                options={(item.data as Edge).transportOptions}
                                selectedOptionIndex={selectedIdx}
                                onSelectOption={(i) => handleSelect(id, i)}
                            />
                        )}
                    </Box>
                )
            })}
        </VStack>
    )
}

export const ItineraryView = ({ itinerary, possibleItineraries }: ItineraryViewProps) => {
    const options = possibleItineraries && possibleItineraries.length > 0 ? possibleItineraries : (itinerary ? [itinerary] : [])

    if (options.length === 0) return null

    return (
        <VStack spacing={4} align="stretch" w="full">
            <Box textAlign="left">
                <Heading size="lg" fontFamily="'Merriweather', serif">{options[0].title}</Heading>
                <Text color="gray.400">{options[0].description}</Text>
            </Box>

            <Divider borderColor="whiteAlpha.200" />

            {options.length > 1 ? (
                <Tabs variant="soft-rounded" colorScheme="green" align="center">
                    <TabList mb={4} bg="#111827" p={1} borderRadius="xl" border="1px solid" borderColor="whiteAlpha.100" width="fit-content" mx="auto">
                        {options.map((_, idx) => (
                            <Tab key={idx} color="gray.400" _selected={{ color: 'white', bg: 'green.600' }} borderRadius="lg" fontSize="md">
                                {options[idx].title || `Option ${idx + 1}`}
                            </Tab>
                        ))}
                    </TabList>
                    <TabPanels>
                        {options.map((it, idx) => (
                            <TabPanel key={idx} px={0}>
                                <SingleItineraryTimeline itinerary={it} />
                            </TabPanel>
                        ))}
                    </TabPanels>
                </Tabs>
            ) : (
                <SingleItineraryTimeline itinerary={options[0]} />
            )}
        </VStack>
    )
}
