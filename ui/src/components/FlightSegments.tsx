import { Box, HStack, Icon, Badge, Text, VStack } from "@chakra-ui/react";
import { MdFlightTakeoff, MdFlightLand } from "react-icons/md";

interface Layover {
  airportCode: string;
  duration: string;
  arrivalTime?: Date;
  departureTime?: Date;
}

interface FlightSegment {
  carrierCode: string;
  flightNumber: string;
  departureAirportCode: string;
  arrivalAirportCode: string;
  departureTime?: { toDate: () => Date };
  arrivalTime?: { toDate: () => Date };
  duration?: string;
}

interface FlightSegmentsProps {
  segments: FlightSegment[];
  layovers: Layover[];
}

export const FlightSegments = ({ segments, layovers }: FlightSegmentsProps) => {
  return (
    <Box>
      <HStack spacing={2} color="blue.300" mb={3}>
        <Icon as={MdFlightTakeoff} boxSize={5} />
        <Text fontSize="md" fontWeight="semibold">
          Flight Details & Layovers
        </Text>
      </HStack>
      <VStack align="start" spacing={3} ml={6}>
        {segments.map((seg: FlightSegment, idx: number) => {
          const depTime = seg.departureTime?.toDate();
          const arrTime = seg.arrivalTime?.toDate();
          const timeStr =
            depTime && arrTime
              ? `${depTime.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })} → ${arrTime.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}`
              : "";

          return (
            <Box key={idx}>
              <HStack align="center" spacing={3}>
                <Badge colorScheme="blue" variant="subtle" fontSize="sm">
                  {seg.carrierCode} {seg.flightNumber}
                </Badge>
                <Text fontSize="sm" color="gray.300">
                  {seg.departureAirportCode} → {seg.arrivalAirportCode}
                </Text>
                {timeStr && (
                  <Text fontSize="sm" color="gray.400">
                    {timeStr}
                  </Text>
                )}
                {seg.duration && (
                  <Text fontSize="sm" color="gray.500">
                    ({seg.duration})
                  </Text>
                )}
              </HStack>

              {/* Show layover info after each segment except the last */}
              {idx < segments.length - 1 && layovers[idx] && (
                <Box mt={4}>
                  <LayoverInfo layover={layovers[idx]} />
                </Box>
              )}
            </Box>
          );
        })}
      </VStack>
    </Box>
  );
};

interface LayoverInfoProps {
  layover: Layover;
}

const LayoverInfo = ({ layover }: LayoverInfoProps) => {
  // Format duration from PT9H40M format to readable format or use time difference
  const formatDuration = (
    duration: string,
    arrivalTime?: Date,
    departureTime?: Date,
  ) => {
    // If we have actual times, calculate the real duration
    if (arrivalTime && departureTime) {
      const diffMs = departureTime.getTime() - arrivalTime.getTime();
      const hours = Math.floor(diffMs / (1000 * 60 * 60));
      const minutes = Math.floor((diffMs % (1000 * 60 * 60)) / (1000 * 60));
      return `${hours}h ${minutes}m`;
    }

    // Parse ISO 8601 duration format (e.g., PT9H40M)
    const match = duration.match(/PT(?:(\d+)H)?(?:(\d+)M)?/);
    if (match) {
      const hours = match[1] || "0";
      const minutes = match[2] || "0";
      return `${hours}h ${minutes}m`;
    }

    return duration;
  };

  // Format timezone information
  const formatTimezoneInfo = (arrivalTime?: Date, departureTime?: Date) => {
    if (arrivalTime && departureTime) {
      const arrTz = arrivalTime
        .toLocaleTimeString([], { timeZoneName: "short" })
        .split(" ")
        .pop();
      const depTz = departureTime
        .toLocaleTimeString([], { timeZoneName: "short" })
        .split(" ")
        .pop();
      return `${arrTz} → ${depTz}`;
    }
    return null;
  };

  const durationStr = formatDuration(
    layover.duration,
    layover.arrivalTime,
    layover.departureTime,
  );
  const timezoneStr = formatTimezoneInfo(
    layover.arrivalTime,
    layover.departureTime,
  );

  return (
    <HStack ml={4} mt={3} spacing={3} align="center">
      <Box height="24px" width="2px" bg="gray.600" />
      <Icon as={MdFlightLand} boxSize={4} color="orange.400" />
      <Text fontSize="sm" color="orange.300" fontWeight="medium">
        Layover: {layover.airportCode}
      </Text>
      <Text fontSize="sm" color="gray.300">
        {durationStr}
      </Text>
      {timezoneStr && (
        <Text fontSize="sm" color="gray.400">
          ({timezoneStr})
        </Text>
      )}
    </HStack>
  );
};
