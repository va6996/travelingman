import {
    VStack,
    HStack,
    Box,
    Icon,
    Text,
    Tooltip,
    Badge,
    Divider
} from '@chakra-ui/react'
import { MdLuggage, MdCheckCircle, MdWarning, MdInfo } from 'react-icons/md'
import { Transport, BaggageType } from '../gen/protos/itinerary_pb'

interface BaggageDisplayProps {
    transport: Transport
    showTitle?: boolean
    compact?: boolean
}

// Helper to get comprehensive baggage info
const getBaggageInfo = (transport: Transport) => {
    if (transport.details.case !== 'flight') return null

    const flight = transport.details.value
    const baggagePolicies = flight.baggagePolicy || []

    if (baggagePolicies.length === 0) {
        return { hasPolicy: false, policies: [] }
    }

    const policies = baggagePolicies.map((policy: any) => ({
        type: policy.type,
        quantity: policy.quantity,
        weight: policy.weight,
        unit: policy.weightUnit
    }))

    return { hasPolicy: true, policies }
}

// Helper to get baggage-related ancillary costs
const getBaggageAncillaries = (transport: Transport) => {
    if (transport.details.case !== 'flight') return []

    const flight = transport.details.value
    const ancillaries = flight.ancillaryCosts || []
    
    return ancillaries.filter((ancillary: any) => 
        ancillary.type?.toLowerCase().includes('baggage') || 
        ancillary.description?.toLowerCase().includes('bag') ||
        ancillary.description?.toLowerCase().includes('luggage')
    )
}

export const BaggageDisplay = ({ transport, showTitle = true, compact = false }: BaggageDisplayProps) => {
    const baggageInfo = getBaggageInfo(transport)
    const baggageAncillaries = getBaggageAncillaries(transport)

    if (!baggageInfo || !baggageInfo.hasPolicy) {
        return null
    }

    const checkedBags = baggageInfo.policies.find(p => p.type === BaggageType.CHECKED)
    const carryonBags = baggageInfo.policies.find(p => p.type === BaggageType.CARRYON)

    if (compact) {
        return (
            <HStack spacing={2} color="blue.300">
                <Icon as={MdLuggage} boxSize={4} />
                <VStack align="start" spacing={0}>
                    {checkedBags && checkedBags.quantity > 0 && (
                        <Text fontSize="xs" color="gray.300">
                            {checkedBags.quantity} checked{checkedBags.weight > 0 && ` (${checkedBags.weight}${checkedBags.unit})`}
                        </Text>
                    )}
                    {carryonBags && carryonBags.quantity > 0 && (
                        <Text fontSize="xs" color="gray.300">
                            {carryonBags.quantity} carry-on{carryonBags.weight > 0 && ` (${carryonBags.weight}${carryonBags.unit})`}
                        </Text>
                    )}
                </VStack>
                {baggageAncillaries.length > 0 && (
                    <Badge colorScheme="orange" variant="solid" fontSize="xs">
                        +{baggageAncillaries.length} extra
                    </Badge>
                )}
            </HStack>
        )
    }

    return (
        <Box>
            {showTitle && (
                <HStack spacing={2} color="blue.300" mb={2}>
                    <Icon as={MdLuggage} boxSize={4} />
                    <Text fontSize="sm" fontWeight="semibold">Baggage Allowance</Text>
                    <Tooltip label="Baggage policies may vary by airline and ticket type">
                        <Icon as={MdInfo} boxSize={3} color="gray.400" cursor="help" />
                    </Tooltip>
                </HStack>
            )}
            
            <VStack align="start" spacing={2} ml={showTitle ? 6 : 0}>
                {checkedBags && checkedBags.quantity > 0 && (
                    <HStack spacing={2}>
                        <Icon as={MdCheckCircle} boxSize={3} color="green.400" />
                        <Text fontSize="sm" color="gray.300">
                            {checkedBags.quantity} checked bag{checkedBags.quantity > 1 ? 's' : ''} included
                            {checkedBags.weight > 0 && ` (${checkedBags.weight}${checkedBags.unit})`}
                        </Text>
                    </HStack>
                )}
                
                {carryonBags && carryonBags.quantity > 0 && (
                    <HStack spacing={2}>
                        <Icon as={MdCheckCircle} boxSize={3} color="green.400" />
                        <Text fontSize="sm" color="gray.300">
                            {carryonBags.quantity} carry-on bag{carryonBags.quantity > 1 ? 's' : ''} included
                            {carryonBags.weight > 0 && ` (${carryonBags.weight}${carryonBags.unit})`}
                        </Text>
                    </HStack>
                )}

                {!checkedBags && !carryonBags && (
                    <HStack spacing={2}>
                        <Icon as={MdWarning} boxSize={3} color="yellow.400" />
                        <Text fontSize="sm" color="yellow.300">
                            No checked baggage included
                        </Text>
                    </HStack>
                )}
            </VStack>

            {baggageAncillaries.length > 0 && (
                <>
                    <Divider borderColor="gray.600" my={2} />
                    <HStack spacing={2} color="orange.300">
                        <Icon as={MdWarning} boxSize={3} />
                        <Text fontSize="xs" fontWeight="semibold">Additional Baggage Fees</Text>
                    </HStack>
                    <VStack align="start" spacing={1} ml={6}>
                        {baggageAncillaries.map((ancillary: any, idx: number) => (
                            <HStack key={idx} justify="space-between" width="100%">
                                <Text fontSize="xs" color="gray.400">
                                    {ancillary.description || ancillary.type}
                                </Text>
                                <Text fontSize="xs" color="orange.200" fontWeight="semibold">
                                    +{ancillary.cost?.value?.toFixed(2)} {ancillary.cost?.currency}
                                </Text>
                            </HStack>
                        ))}
                    </VStack>
                </>
            )}
        </Box>
    )
}

export default BaggageDisplay