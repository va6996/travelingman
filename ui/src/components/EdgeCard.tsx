import {
    Text,
    VStack,
    HStack,
    Box,
    Icon,
    Badge
} from '@chakra-ui/react'
import { MdSwapHoriz } from 'react-icons/md'
import { Transport, TransportType } from '../gen/protos/itinerary_pb'
import { TimelineCard, OptionCard } from './TimelineCard'

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

    return (
        <TimelineCard
            themeColor="green"
            title={getTitle()}
            subtitle={typeof getSortDetails() === 'string' ? <Text fontSize="sm" color="gray.400">{getSortDetails()}</Text> : getSortDetails()}
            price={currentTransport.cost?.value ? `${currentTransport.cost.value.toFixed(2)} ${currentTransport.cost.currency}` : undefined}
            tags={currentTransport.tags}
            hideToggle={options.length <= 1}
        >
            <VStack align="stretch" spacing={4}>
                {options.length > 1 && (
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
                                            </VStack>
                                            <VStack>
                                                {isSelected && <Badge colorScheme="green" variant="solid">Selected</Badge>}
                                                <Text fontWeight="bold" color="yellow.200" fontSize="md">
                                                    {opt.cost?.value} {opt.cost?.currency}
                                                </Text>
                                            </VStack>
                                        </HStack>
                                    </OptionCard>
                                )
                            })}
                        </VStack>
                    </Box>
                )}
            </VStack>
        </TimelineCard>
    )
}

function capitalizeFirstLetter(val: string) {
    val = val.toLowerCase()
    return val.replace(/\b\w/g, l => l.toUpperCase());
}