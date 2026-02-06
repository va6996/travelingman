import {
    VStack,
    HStack,
    Box,
    Icon,
    Text,
    Badge,
    Divider,
    Tooltip,
    Accordion,
    AccordionItem,
    AccordionButton,
    AccordionPanel,
    AccordionIcon
} from '@chakra-ui/react'
import { MdLuggage, MdCheckCircle, MdInfo } from 'react-icons/md'
import { Itinerary } from '../gen/protos/graph_pb'
import { BaggageType } from '../gen/protos/itinerary_pb'

interface ItineraryBaggageSummaryProps {
    itinerary: Itinerary
}

// Helper to get baggage info from a transport
const getTransportBaggageInfo = (transport: any) => {
    if (transport?.details?.case !== 'flight') return null

    const flight = transport.details.value
    const baggagePolicies = flight.baggagePolicy || []

    const checkedBags = baggagePolicies.find((p: any) => p.type === BaggageType.CHECKED)
    const carryonBags = baggagePolicies.find((p: any) => p.type === BaggageType.CARRYON)
    const baggageAncillaries = flight.ancillaryCosts?.filter((ancillary: any) =>
        ancillary.type?.toLowerCase().includes('baggage') ||
        ancillary.description?.toLowerCase().includes('bag') ||
        ancillary.description?.toLowerCase().includes('luggage')
    ) || []

    return {
        checkedBags,
        carryonBags,
        baggageAncillaries,
        route: `${transport.originLocation?.city || 'Origin'} → ${transport.destinationLocation?.city || 'Destination'}`,
        flightInfo: `${flight.carrierCode} ${flight.flightNumber}`
    }
}

export const ItineraryBaggageSummary = ({ itinerary }: ItineraryBaggageSummaryProps) => {
    if (!itinerary.graph?.edges) return null

    // Collect all baggage information from flight edges
    const flightBaggageInfo = itinerary.graph.edges
        .map(edge => getTransportBaggageInfo(edge.transport))
        .filter(info => info !== null)

    if (flightBaggageInfo.length === 0) return null

    // Calculate totals
    const totalCheckedBags = flightBaggageInfo.reduce((sum, info) =>
        sum + (info.checkedBags?.quantity || 0), 0)
    const totalCarryonBags = flightBaggageInfo.reduce((sum, info) =>
        sum + (info.carryonBags?.quantity || 0), 0)
    const totalAncillaryCosts = flightBaggageInfo.reduce((sum, info) =>
        sum + info.baggageAncillaries.reduce((ancSum: number, anc: any) =>
            ancSum + (anc.cost?.value || 0), 0), 0)

    const currency = flightBaggageInfo[0]?.baggageAncillaries[0]?.cost?.currency || 'USD'

    return (
        <Box bg="gray.800" borderRadius="md" p={4} mb={6}>
            <HStack spacing={2} color="blue.300" mb={4}>
                <Icon as={MdLuggage} boxSize={5} />
                <Text fontSize="md" fontWeight="bold">Baggage Summary</Text>
                <Tooltip label="Baggage policies may vary by airline and ticket type">
                    <Icon as={MdInfo} boxSize={4} color="gray.400" cursor="help" />
                </Tooltip>
            </HStack>

            {/* Quick Overview */}
            <VStack align="start" spacing={2} mb={4}>
                <HStack spacing={4}>
                    {totalCheckedBags > 0 && (
                        <HStack spacing={1}>
                            <Icon as={MdCheckCircle} boxSize={3} color="green.400" />
                            <Text fontSize="sm" color="gray.300">
                                {totalCheckedBags} checked bag{totalCheckedBags > 1 ? 's' : ''} included
                            </Text>
                        </HStack>
                    )}
                    {totalCarryonBags > 0 && (
                        <HStack spacing={1}>
                            <Icon as={MdCheckCircle} boxSize={3} color="green.400" />
                            <Text fontSize="sm" color="gray.300">
                                {totalCarryonBags} carry-on bag{totalCarryonBags > 1 ? 's' : ''} included
                            </Text>
                        </HStack>
                    )}

                </HStack>

                {totalAncillaryCosts > 0 && (
                    <HStack spacing={2}>
                        <Badge colorScheme="orange" variant="solid" fontSize="xs">
                            Additional fees: {totalAncillaryCosts.toFixed(2)} {currency}
                        </Badge>
                    </HStack>
                )}
            </VStack>

            {/* Detailed Breakdown */}
            <Accordion allowToggle>
                <AccordionItem border="none">
                    <AccordionButton px={0} py={2}>
                        <Box flex="1" textAlign="left">
                            <Text fontSize="sm" color="gray.400" fontWeight="semibold">
                                View detailed breakdown ({flightBaggageInfo.length} flight{flightBaggageInfo.length > 1 ? 's' : ''})
                            </Text>
                        </Box>
                        <AccordionIcon />
                    </AccordionButton>
                    <AccordionPanel px={0} pb={4}>
                        <VStack align="start" spacing={3}>
                            {flightBaggageInfo.map((info, idx) => (
                                <Box key={idx} w="full">
                                    <HStack justify="space-between" mb={2}>
                                        <Text fontSize="sm" fontWeight="semibold" color="gray.300">
                                            {info.flightInfo}
                                        </Text>
                                        <Text fontSize="xs" color="gray.500">
                                            {info.route}
                                        </Text>
                                    </HStack>

                                    <VStack align="start" spacing={1} ml={4}>
                                        {info.checkedBags && info.checkedBags.quantity > 0 && (
                                            <Text fontSize="xs" color="gray.400">
                                                • {info.checkedBags.quantity} checked bag{info.checkedBags.quantity > 1 ? 's' : ''}
                                                {info.checkedBags.weight > 0 && ` (${info.checkedBags.weight}${info.checkedBags.weightUnit})`}
                                            </Text>
                                        )}
                                        {info.carryonBags && info.carryonBags.quantity > 0 && (
                                            <Text fontSize="xs" color="gray.400">
                                                • {info.carryonBags.quantity} carry-on bag{info.carryonBags.quantity > 1 ? 's' : ''}
                                                {info.carryonBags.weight > 0 && ` (${info.carryonBags.weight}${info.carryonBags.weightUnit})`}
                                            </Text>
                                        )}
                                        {info.baggageAncillaries.length > 0 && (
                                            <VStack align="start" spacing={1} ml={2}>
                                                {info.baggageAncillaries.map((ancillary: any, ancIdx: number) => (
                                                    <HStack key={ancIdx} justify="space-between" w="90%">
                                                        <Text fontSize="xs" color="orange.300">
                                                            + {ancillary.description || ancillary.type}
                                                        </Text>
                                                        <Text fontSize="xs" color="orange.200" fontWeight="semibold">
                                                            {ancillary.cost?.value?.toFixed(2)} {ancillary.cost?.currency}
                                                        </Text>
                                                    </HStack>
                                                ))}
                                            </VStack>
                                        )}
                                    </VStack>

                                    {idx < flightBaggageInfo.length - 1 && <Divider borderColor="gray.700" mt={3} />}
                                </Box>
                            ))}
                        </VStack>
                    </AccordionPanel>
                </AccordionItem>
            </Accordion>
        </Box>
    )
}

export default ItineraryBaggageSummary