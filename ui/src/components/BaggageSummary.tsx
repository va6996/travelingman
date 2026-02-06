import {
    HStack,
    VStack,
    Icon,
    Text,
    Tooltip,
    Badge
} from '@chakra-ui/react'
import { MdLuggage, MdWarning, MdWorkOutline } from 'react-icons/md'
import { Transport, BaggageType } from '../gen/protos/itinerary_pb'

interface BaggageSummaryProps {
    transport: Transport
}

export const BaggageSummary = ({ transport }: BaggageSummaryProps) => {
    if (transport.details.case !== 'flight') return null

    const flight = transport.details.value
    const baggagePolicies = flight.baggagePolicy || []

    if (baggagePolicies.length === 0) {
        return (
            <Tooltip label="No baggage information available">
                <HStack spacing={1}>
                    <Icon as={MdLuggage} boxSize={3} color="gray.400" />
                    <Text fontSize="xs" color="gray.400">No info</Text>
                </HStack>
            </Tooltip>
        )
    }

    const checkedBags = baggagePolicies.find((p: any) => p.type === BaggageType.CHECKED)
    const carryonBags = baggagePolicies.find((p: any) => p.type === BaggageType.CARRYON)

    // Count total included bags
    const totalIncludedBags = (checkedBags?.quantity || 0) + (carryonBags?.quantity || 0)

    // Check for baggage ancillary costs
    const baggageAncillaries = flight.ancillaryCosts?.filter((ancillary: any) =>
        ancillary.type?.toLowerCase().includes('baggage') ||
        ancillary.description?.toLowerCase().includes('bag') ||
        ancillary.description?.toLowerCase().includes('luggage')
    ) || []

    if (totalIncludedBags === 0 && baggageAncillaries.length === 0) {
        return (
            <Tooltip label="No checked baggage included">
                <HStack spacing={1}>
                    <Icon as={MdWarning} boxSize={3} color="yellow.400" />
                    <Text fontSize="xs" color="yellow.400">No bags</Text>
                </HStack>
            </Tooltip>
        )
    }

    return (
        <VStack align="start" spacing={0.5}>
            {carryonBags && carryonBags.quantity > 0 && (
                <HStack spacing={1}>
                    <Icon as={MdWorkOutline} boxSize={3} color="blue.400" />
                    <Text fontSize="xs" color="gray.300">
                        {carryonBags.quantity} carry-on
                    </Text>
                </HStack>
            )}
            {checkedBags && checkedBags.quantity > 0 && (
                <HStack spacing={1}>
                    <Icon as={MdLuggage} boxSize={3} color="blue.400" />
                    <Text fontSize="xs" color="gray.300">
                        {checkedBags.quantity} checked
                    </Text>
                </HStack>
            )}
            {baggageAncillaries.length > 0 && (
                <Badge colorScheme="orange" variant="solid" fontSize="xs">
                    +{baggageAncillaries.length} extra
                </Badge>
            )}
        </VStack>
    )
}

export default BaggageSummary