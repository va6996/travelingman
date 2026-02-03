import {
    Text,
    VStack,
    HStack,
    Box,
    Icon,
    Badge,
    Divider,
    Wrap,
    WrapItem
} from '@chakra-ui/react'
import { MdSwapHoriz, MdLuggage, MdAddCircle } from 'react-icons/md'
import { Transport, TransportType } from '../gen/protos/itinerary_pb'
import { TimelineCard, OptionCard } from './TimelineCard'

// Helper to get baggage info display
const getBaggageInfo = (transport: Transport) => {
    if (transport.details.case !== 'flight') return null

    const flight = transport.details.value
    const baggagePolicies = flight.baggagePolicy || []

    if (baggagePolicies.length === 0) {
        return null
    }

    // Get checked bags info (BAGGAGE_TYPE_CHECKED = 2)
    const checkedBags = baggagePolicies.find((p: any) => p.type === 2)
    if (checkedBags && checkedBags.quantity > 0) {
        return `${checkedBags.quantity} checked bag${checkedBags.quantity > 1 ? 's' : ''} included (${checkedBags.weight}${checkedBags.weightUnit})`
    }

    return null
}

// Helper to get ancillary costs display
const getAncillaryCosts = (transport: Transport) => {
    if (transport.details.case !== 'flight') return null

    const flight = transport.details.value
    const ancillaries = flight.ancillaryCosts || []

    if (ancillaries.length === 0) {
        return null
    }

    return ancillaries
}

// Helper to get total cost with ancillaries
const getTotalCost = (transport: Transport) => {
    if (transport.details.case !== 'flight') return null

    const flight = transport.details.value
    const totalWithAncillaries = flight.totalCostWithAncillaries

    if (totalWithAncillaries && totalWithAncillaries.value) {
        return totalWithAncillaries
    }

    return transport.cost
}

interface EdgeCardProps {
    transport: Transport
    options: Transport[]
    selectedOptionIndex: number
    onSelectOption: (index: number) => void
}

export const EdgeCard = ({ transport, options, selectedOptionIndex, onSelectOption }: EdgeCardProps) => {
    const currentTransport = options.length > 0 ? options[selectedOptionIndex] : transport

    // Helper to safely get details
    const getSortDetails = () => {
        let timing = ""
        let duration = ""

        if (currentTransport.details.case === 'flight') {
            const f = currentTransport.details.value;
            const dep = f.departureTime?.toDate();
            const arr = f.arrivalTime?.toDate();
            if (dep && arr) {
                timing = `${dep.toLocaleDateString([], { month: 'short', day: 'numeric' })}  ${dep.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })} - ${arr.toLocaleDateString([], { month: 'short', day: 'numeric' })}  ${arr.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}`
                const travelTime = arr.getTime() - dep.getTime();
                const hours = Math.floor(travelTime / (1000 * 60 * 60));
                const minutes = Math.floor((travelTime % (1000 * 60 * 60)) / (1000 * 60));
                duration = `${hours}h ${minutes}m`;
            }
        }

        const origin = capitalizeFirstLetter(currentTransport.originLocation?.city || "Origin")
        const dest = capitalizeFirstLetter(currentTransport.destinationLocation?.city || "Dest")

        return (
            <VStack align="start" spacing={0}>
                <Text fontSize="md" fontWeight="bold">{origin} ➝ {dest}</Text>
                {timing && <Text fontSize="md" color="gray.400">{timing}</Text>}
                {duration && <Text fontSize="md" color="gray.400">Travel Time: {duration}</Text>}
            </VStack>
        )
    }

    // Build title for flights
    const getTitle = () => {
        if (currentTransport.type === TransportType.FLIGHT && currentTransport.details.case === 'flight') {
            const f = currentTransport.details.value
            return (
                <HStack spacing={2}>
                    <Text>Flight</Text>
                    <Text>{f.carrierCode} {f.flightNumber}</Text>
                </HStack>
            )
        }
        return currentTransport.type === TransportType.FLIGHT ? "Flight" : "Travel"
    }

    const baggageInfo = getBaggageInfo(currentTransport)
    const ancillaryCosts = getAncillaryCosts(currentTransport)
    const totalCost = getTotalCost(currentTransport)
    const baseCost = currentTransport.cost?.value ? `${currentTransport.cost.value.toFixed(2)} ${currentTransport.cost.currency}` : undefined
    const totalCostDisplay = totalCost?.value ? `${totalCost.value.toFixed(2)} ${totalCost.currency}` : baseCost

    return (
        <TimelineCard
            themeColor="green"
            title={getTitle()}
            subtitle={typeof getSortDetails() === 'string' ? <Text fontSize="sm" color="gray.400">{getSortDetails()}</Text> : getSortDetails()}
            price={totalCostDisplay}
            tags={currentTransport.tags}
            hideToggle={options.length <= 1 || (!!ancillaryCosts && ancillaryCosts.length > 0)}
        >
            <VStack align="stretch" spacing={4}>
                {/* Baggage Information */}
                {baggageInfo && (
                    <Box>
                        <HStack spacing={2} color="blue.300">
                            <Icon as={MdLuggage} boxSize={4} />
                            <Text fontSize="sm" fontWeight="semibold">Baggage Allowance</Text>
                        </HStack>
                        <Text fontSize="sm" color="gray.300" ml={6}>{baggageInfo}</Text>
                    </Box>
                )}

                {/* Ancillary Costs (Extra Bags, etc.) */}
                {ancillaryCosts && ancillaryCosts.length > 0 && (
                    <Box>
                        <HStack spacing={2} color="orange.300">
                            <Icon as={MdAddCircle} boxSize={4} />
                            <Text fontSize="sm" fontWeight="semibold">Additional Services</Text>
                        </HStack>
                        <VStack align="start" spacing={1} ml={6}>
                            {ancillaryCosts.map((ancillary: any, idx: number) => (
                                <HStack key={idx} justify="space-between" width="100%">
                                    <Text fontSize="sm" color="gray.300">
                                        {ancillary.description || ancillary.type}
                                    </Text>
                                    <Text fontSize="sm" color="yellow.200" fontWeight="semibold">
                                        +{ancillary.cost?.value?.toFixed(2)} {ancillary.cost?.currency}
                                    </Text>
                                </HStack>
                            ))}
                            <Divider borderColor="gray.600" my={1} />
                            <HStack justify="space-between" width="100%">
                                <Text fontSize="xs" color="gray.400">Base Fare</Text>
                                <Text fontSize="xs" color="gray.400">{baseCost}</Text>
                            </HStack>
                            <HStack justify="space-between" width="100%">
                                <Text fontSize="sm" color="white" fontWeight="bold">Total with Extras</Text>
                                <Text fontSize="sm" color="green.300" fontWeight="bold">{totalCostDisplay}</Text>
                            </HStack>
                        </VStack>
                    </Box>
                )}

                {/* Flight Options */}
                {options.length > 1 && (
                    <>
                        <Divider borderColor="gray.600" />
                        <Box>
                            <HStack mb={2}>
                                <Icon as={MdSwapHoriz} color="gray.400" />
                                <Text fontWeight="semibold" color="gray.300" fontSize="s">Available Options</Text>
                            </HStack>
                            <VStack align="stretch" spacing={2}>
                                {options.map((opt, idx) => {
                                    const isSelected = idx === selectedOptionIndex;
                                    // Calc duration and timing
                                    let durationStr = "N/A"
                                    let timing = ""
                                    if (opt.details.case === 'flight') {
                                        const f = opt.details.value
                                        if (f.departureTime && f.arrivalTime) {
                                            const diff = f.arrivalTime.seconds - f.departureTime.seconds
                                            const h = Math.floor(Number(diff) / 3600)
                                            const m = Math.floor((Number(diff) % 3600) / 60)
                                            durationStr = `${h}h ${m}m`

                                            const dep = f.departureTime.toDate()
                                            const arr = f.arrivalTime.toDate()
                                            timing = `${dep.toLocaleDateString([], { month: 'short', day: 'numeric' })}  ${dep.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })} - ${arr.toLocaleDateString([], { month: 'short', day: 'numeric' })}  ${arr.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}`
                                        }
                                    }

                                    // Get ancillary costs for this option
                                    const optAncillaries = opt.details.case === 'flight' ? getAncillaryCosts(opt) : null
                                    const optTotalCost = opt.details.case === 'flight' ? getTotalCost(opt) : null
                                    const optPrice = optTotalCost?.value ?
                                        `${optTotalCost.value.toFixed(2)} ${optTotalCost.currency}` :
                                        (opt.cost?.value ? `${opt.cost.value.toFixed(2)} ${opt.cost.currency}` : '')

                                    return (
                                        <OptionCard
                                            key={idx}
                                            isSelected={isSelected}
                                            themeColor="green"
                                            onSelect={() => onSelectOption(idx)}
                                        >
                                            <HStack justify="space-between">
                                                <VStack align="start" spacing={0}>
                                                    <Text fontWeight="bold" fontSize="md" color="white">
                                                        {opt.details.case === 'flight' ? `${opt.details.value.carrierCode} ${opt.details.value.flightNumber}` : opt.plugin}
                                                    </Text>
                                                    <Text fontSize="sm" color="gray.400">{timing}</Text>
                                                    <Text fontSize="sm" color="gray.500">Duration: {durationStr}</Text>
                                                    <Text fontSize="sm" color="gray.500">{opt.tags.join(", ")}</Text>
                                                    {optAncillaries && optAncillaries.length > 0 && (
                                                        <Wrap>
                                                            {optAncillaries.map((anc: any, ancIdx: number) => (
                                                                <WrapItem key={ancIdx}>
                                                                    <Badge colorScheme="orange" variant="subtle" fontSize="xs">
                                                                        +{anc.description}
                                                                    </Badge>
                                                                </WrapItem>
                                                            ))}
                                                        </Wrap>
                                                    )}
                                                </VStack>
                                                <VStack>
                                                    {isSelected && <Badge colorScheme="green" variant="solid">Selected</Badge>}
                                                    <Text fontWeight="bold" color="yellow.200" fontSize="md">
                                                        {optPrice}
                                                    </Text>
                                                </VStack>
                                            </HStack>
                                        </OptionCard>
                                    )
                                })}
                            </VStack>
                        </Box>
                    </>
                )}
            </VStack>
        </TimelineCard>
    )
}

function capitalizeFirstLetter(val: string) {
    val = val.toLowerCase()
    return val.replace(/\b\w/g, l => l.toUpperCase());
}