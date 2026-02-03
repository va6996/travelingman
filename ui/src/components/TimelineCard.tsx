import {
    Box,
    Text,
    Badge,
    VStack,
    HStack,
    Collapse,
    useDisclosure,
    Icon,
    BoxProps
} from '@chakra-ui/react'
import { MdKeyboardArrowDown, MdKeyboardArrowUp } from 'react-icons/md'

interface TimelineCardProps {
    themeColor: string // e.g. "purple", "blue"
    title: React.ReactNode
    subtitle: React.ReactNode
    price?: string
    rightContent?: React.ReactNode
    tags: string[]
    children: React.ReactNode // Expanded content
    isExpandedDefault?: boolean
    hideToggle?: boolean
}

export const TimelineCard = ({
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

    const style = {
        '--theme-color': `var(--chakra-colors-${themeColor}-500)`,
        '--theme-color-900': `var(--chakra-colors-${themeColor}-900)`
    } as React.CSSProperties

    return (
        <Box
            className="timeline-card"
            style={style}
        >
            <Box p={4} pb={2}>
                <HStack justify="space-between" align="start">
                    <VStack align="start" spacing={1}>
                        <Text fontWeight="bold" fontSize="xl" color="whiteAlpha.900" lineHeight="shorter">
                            {title}
                        </Text>
                        {subtitle && (
                            <Box as="div" fontSize="lg" color="whiteAlpha.700">
                                {subtitle}
                            </Box>
                        )}
                    </VStack>
                    <VStack align="end" spacing={0}>
                        {price && (
                            <Text fontWeight="bold" color="yellow.200" fontSize="lg">
                                {price}
                            </Text>
                        )}
                        {rightContent}
                    </VStack>
                </HStack>

                {(tags.length > 0 || !hideToggle) && (
                    <HStack justify="space-between" align="center" mt={3} pb={hideToggle && tags.length > 0 ? 2 : 0}>
                        {tags.length > 0 ? (
                            <HStack spacing={2} flexWrap="wrap">
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
                        ) : (
                            <Box />
                        )}

                        {!hideToggle && (
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
                                flexShrink={0}
                            >
                                <Icon as={isOpen ? MdKeyboardArrowUp : MdKeyboardArrowDown} boxSize={5} />
                            </Box>
                        )}
                    </HStack>
                )}
            </Box>

            <Collapse in={isOpen} animateOpacity>
                <Box className="timeline-card-options" p={4} pt={3}>
                    {children}
                </Box>
            </Collapse>
        </Box>
    )
}

interface OptionCardProps extends BoxProps {
    isSelected: boolean
    themeColor: string
    onSelect: () => void
    children: React.ReactNode
}

export const OptionCard = ({ isSelected, themeColor, onSelect, children, ...props }: OptionCardProps) => {
    const style = {
        '--theme-color': `var(--chakra-colors-${themeColor}-500)`,
        '--theme-color-900': `var(--chakra-colors-${themeColor}-900)`
    } as React.CSSProperties

    return (
        <Box
            className={`option-card ${isSelected ? 'selected' : ''}`}
            style={style}
            onClick={onSelect}
            {...props}
        >
            {children}
        </Box>
    )
}
