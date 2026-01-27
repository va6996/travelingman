import {
    Text,
    VStack,
    HStack,
    Box,
    Icon,
    Badge
} from '@chakra-ui/react'
import { MdSwapHoriz } from 'react-icons/md'
import { Accommodation } from '../gen/protos/itinerary_pb'
import { TimelineCard, OptionCard } from './TimelineCard'


interface NodeCardProps {
    nodeId: string
    stay: Accommodation
    options: Accommodation[]
    selectedOptionIndex: number
    onSelectOption: (index: number) => void
    locationName?: string
}

export const NodeCard = ({ stay, options, selectedOptionIndex, onSelectOption, locationName }: NodeCardProps) => {
    const currentStay = options.length > 0 ? options[selectedOptionIndex] : stay

    // If no stay data, treat as a location/waypoint
    if (!currentStay) {
        return (
            <TimelineCard
                themeColor="gray"
                title={locationName || "Location / Waypoint"}
                subtitle="No accommodation booked"
                tags={[]}
                hideToggle={true}
            >
                <Text color="gray.500" fontSize="md">No details available.</Text>
            </TimelineCard>
        )
    }

    const checkIn = currentStay.checkIn ? new Date(Number(currentStay.checkIn.seconds) * 1000) : null
    const checkOut = currentStay.checkOut ? new Date(Number(currentStay.checkOut.seconds) * 1000) : null

    return (
        <TimelineCard
            themeColor="green"
            title={capitalizeFirstLetter(currentStay.name || "Accommodation")}
            subtitle={
                <VStack align="start" spacing={0}>
                    <Text>{currentStay.location?.city ? `${currentStay.location.city}, ${currentStay.location.country || ''}` : (currentStay.address || "Location details unavailable")}</Text>
                    {currentStay.location?.cityCode && (
                        <Text fontSize="sm" color="gray.500">
                            Code: {currentStay.location.cityCode}
                        </Text>
                    )}
                </VStack>
            }
            price={currentStay.cost?.value ? `${currentStay.cost.value.toFixed(2)} ${currentStay.cost.currency}` : ''}
            tags={currentStay.tags}
            hideToggle={options.length <= 1}
            rightContent={
                <>
                    {checkIn && <Text fontSize="sm" color="gray.500">Check In: {checkIn.toLocaleDateString()}</Text>}
                    {checkOut && <Text fontSize="sm" color="gray.500">Check Out: {checkOut.toLocaleDateString()}</Text>}
                    {currentStay.travelerCount && <Text fontSize="sm" color="gray.500">Guests: {currentStay.travelerCount}</Text>}
                </>
            }
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
                                return (
                                    <OptionCard
                                        key={idx}
                                        isSelected={isSelected}
                                        themeColor="green"
                                        onSelect={() => onSelectOption(idx)}
                                    >
                                        <HStack justify="space-between">
                                            <VStack align="start" spacing={0}>
                                                <Text fontWeight="bold" fontSize="md" color="white">{capitalizeFirstLetter(opt.name)}</Text>
                                                <Text fontSize="sm" color="gray.500">{opt.tags.join(", ")}</Text>
                                            </VStack>
                                            <VStack>
                                                {isSelected && <Badge colorScheme="green" variant="solid">Selected</Badge>}
                                                <Text fontWeight="bold" color="yellow.200" fontSize="sm">
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