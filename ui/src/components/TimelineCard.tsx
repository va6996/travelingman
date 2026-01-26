import {
    Box,
    Text,
    Badge,
    VStack,
    HStack,
    Collapse,
    useDisclosure,
    Card,
    CardBody,
    CardHeader,
    Icon,
    BoxProps
} from '@chakra-ui/react'
import { IconType } from 'react-icons'
import { MdKeyboardArrowDown, MdKeyboardArrowUp } from 'react-icons/md'

interface TimelineCardProps {
    icon: IconType
    themeColor: string // e.g. "purple", "blue"
    title: string
    subtitle: React.ReactNode
    price?: string
    rightContent?: React.ReactNode
    tags: string[]
    children: React.ReactNode // Expanded content
    isExpandedDefault?: boolean
    hideToggle?: boolean
}

export const TimelineCard = ({
    icon,
    themeColor,
    title,
    subtitle,
    price,
    rightContent,
    tags,
    children,
    isExpandedDefault = false,
    hideToggle = false
}: TimelineCardProps) => {
    const { isOpen, onToggle } = useDisclosure({ defaultIsOpen: isExpandedDefault })

    return (
        <Card
            variant="filled"
            bg="#111827"
            border="1px solid"
            borderColor="whiteAlpha.100"
            shadow="lg"
            width="full"
            borderRadius="2xl"
            overflow="hidden"
            _hover={{ borderColor: `${themeColor}.500`, shadow: `0 0 0 1px var(--chakra-colors-${themeColor}-500)` }}
            transition="all 0.2s"
            mb={4}
        >
            <CardHeader pb={2}>
                <HStack justify="space-between" align="start">
                    <HStack align="start" spacing={3}>
                        <Box p={2} bg={`${themeColor}.900`} borderRadius="xl" color={`${themeColor}.200`}>
                            <Icon as={icon} boxSize={6} />
                        </Box>
                        <VStack align="start" spacing={1}>
                            <Text fontWeight="bold" fontSize="xl" color="whiteAlpha.900" lineHeight="shorter">
                                {title}
                            </Text>
                            {subtitle && (
                                <Box fontSize="lg" color="whiteAlpha.700">
                                    {subtitle}
                                </Box>
                            )}
                        </VStack>
                    </HStack>
                    <VStack align="end" spacing={0}>
                        {price && (
                            <Text fontWeight="bold" color="yellow.200" fontSize="lg">
                                {price}
                            </Text>
                        )}
                        {rightContent}
                    </VStack>
                </HStack>
            </CardHeader>

            <CardBody pt={2}>
                {tags.length > 0 && (
                    <HStack spacing={2} mb={3} flexWrap="wrap">
                        {tags.map(tag => (
                            <Badge
                                key={tag}
                                colorScheme={themeColor}
                                variant="subtle"
                                fontSize="xs"
                                borderRadius="md"
                                px={2}
                                py={0.5}
                                bg={`${themeColor}.900`}
                                color={`${themeColor}.200`}
                            >
                                {tag}
                            </Badge>
                        ))}
                    </HStack>
                )}

                {!hideToggle && (
                    <HStack justify="flex-end" w="full" mt={-2} mb={2}>
                        <Box
                            as="button"
                            onClick={onToggle}
                            bg="whiteAlpha.200"
                            _hover={{ bg: "whiteAlpha.300" }}
                            p={2}
                            borderRadius="lg"
                            display="flex"
                            alignItems="center"
                            justifyContent="center"
                            color="whiteAlpha.800"
                            fontSize="sm"
                            transition="all 0.2s"
                            aria-label={isOpen ? "Collapse" : "Expand"}
                        >
                            <Icon as={isOpen ? MdKeyboardArrowUp : MdKeyboardArrowDown} boxSize={5} />
                        </Box>
                    </HStack>
                )}

                <Collapse in={isOpen} animateOpacity>
                    <Box mt={3} p={0} bg="transparent" borderRadius="md" fontSize="sm">
                        {children}
                    </Box>
                </Collapse>
            </CardBody>
        </Card>
    )
}

interface OptionCardProps extends BoxProps {
    isSelected: boolean
    themeColor: string
    onSelect: () => void
    children: React.ReactNode
}

export const OptionCard = ({ isSelected, themeColor, onSelect, children, ...props }: OptionCardProps) => {
    return (
        <Box
            p={3}
            borderWidth="1px"
            borderRadius="xl"
            borderColor={isSelected ? `${themeColor}.500` : "whiteAlpha.100"}
            bg={isSelected ? `${themeColor}.900` : "whiteAlpha.50"}
            cursor="pointer"
            onClick={onSelect}
            _hover={{ bg: "whiteAlpha.100" }}
            transition="all 0.2s"
            {...props}
        >
            {children}
        </Box>
    )
}
