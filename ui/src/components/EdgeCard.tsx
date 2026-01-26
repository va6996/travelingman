import {
    Text,
    VStack,
    HStack,
    Box,
    Icon,
    Badge
} from '@chakra-ui/react'
import { MdFlight, MdDirectionsCar, MdTrain, MdDirectionsWalk, MdSwapHoriz } from 'react-icons/md'
import { Transport, TransportType } from '../gen/protos/itinerary_pb'
import { TimelineCard, OptionCard } from './TimelineCard'

interface EdgeCardProps {
    transport: Transport
    options: Transport[]
    selectedOptionIndex: number
    onSelectOption: (index: number) => void
}

const getIcon = (type: TransportType) => {
    switch (type) {
        case TransportType.FLIGHT: return MdFlight;
        case TransportType.CAR: return MdDirectionsCar;
        case TransportType.TRAIN: return MdTrain;
        case TransportType.WALKING: return MdDirectionsWalk;
        default: return MdFlight;
    }
}

export const EdgeCard = ({ transport, options, selectedOptionIndex, onSelectOption }: EdgeCardProps) => {
    const currentTransport = options.length > 0 ? options[selectedOptionIndex] : transport
    const IconComponent = getIcon(currentTransport.type)

    // Helper to safely get details
    const getSortDetails = () => {
        let details = currentTransport.plugin || "Transport"
        let timing = ""

        if (currentTransport.details.case === 'flight') {
            const f = currentTransport.details.value;
            details = `${f.carrierCode} ${f.flightNumber}`
            timing = `${f.departureTime?.toDate().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })} - ${f.arrivalTime?.toDate().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}`
        }

        const origin = currentTransport.originLocation?.iataCodes?.[0] || currentTransport.originLocation?.city || "Origin"
        const dest = currentTransport.destinationLocation?.iataCodes?.[0] || currentTransport.destinationLocation?.city || "Dest"

        return (
            <VStack align="start" spacing={0}>
                <Text fontSize="md" fontWeight="bold">{origin} ➝ {dest}</Text>
                <Text fontSize="md" color="gray.400">{details} • {timing}</Text>
            </VStack>
        )
    }

    return (
        <TimelineCard
            icon={IconComponent}
            themeColor="green"
            title={currentTransport.type === TransportType.FLIGHT ? "Flight" : "Travel"}
            subtitle={typeof getSortDetails() === 'string' ? <Text fontSize="sm" color="gray.400">{getSortDetails()}</Text> : getSortDetails()}
            price={currentTransport.cost?.value ? `${currentTransport.cost.value.toFixed(2)} ${currentTransport.cost.currency}` : undefined}
            tags={currentTransport.tags}
            hideToggle={options.length <= 1}
        >
            <VStack align="stretch" spacing={4}>
                <Text color="gray.300" fontSize="sm"><strong>Ref:</strong> {currentTransport.referenceNumber || "N/A"}</Text>

                {options.length > 1 && (
                    <Box>
                        <HStack mb={2}>
                            <Icon as={MdSwapHoriz} color="gray.400" />
                            <Text fontWeight="semibold" color="gray.300" fontSize="xs">AVAILABLE OPTIONS</Text>
                        </HStack>
                        <VStack align="stretch" spacing={2}>
                            {options.map((opt, idx) => {
                                const isSelected = idx === selectedOptionIndex;
                                // Calc duration and timing
                                let durationStr = "N/A"
                                let timeStr = ""
                                if (opt.details.case === 'flight') {
                                    const f = opt.details.value
                                    if (f.departureTime && f.arrivalTime) {
                                        const diff = f.arrivalTime.seconds - f.departureTime.seconds
                                        const h = Math.floor(Number(diff) / 3600)
                                        const m = Math.floor((Number(diff) % 3600) / 60)
                                        durationStr = `${h}h ${m}m`

                                        const dep = f.departureTime.toDate()
                                        const arr = f.arrivalTime.toDate()
                                        timeStr = `${dep.toLocaleDateString()} ${dep.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })} - ${arr.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}`
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
                                                <Text fontSize="sm" color="gray.400">{timeStr}</Text>
                                                <Text fontSize="sm" color="gray.500">Duration: {durationStr}</Text>
                                                <Text fontSize="sm" color="gray.500">{opt.tags.join(", ")}</Text>
                                            </VStack>
                                            <HStack>
                                                <Text fontWeight="bold" color="yellow.200" fontSize="md">
                                                    {opt.cost?.value} {opt.cost?.currency}
                                                </Text>
                                                {isSelected && <Badge colorScheme="green" variant="solid">Selected</Badge>}
                                            </HStack>
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

