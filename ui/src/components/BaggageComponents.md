# Baggage Information Display Components

This document describes the new UI components for displaying baggage information in the TravelingMan application.

## Components Overview

### 1. `BaggageDisplay` (`components/BaggageDisplay.tsx`)

A comprehensive component that displays detailed baggage information for a single transport.

**Props:**
- `transport: Transport` - The transport object to display baggage info for
- `showTitle?: boolean` - Whether to show the "Baggage Allowance" title (default: true)
- `compact?: boolean` - Whether to show a compact version (default: false)

**Features:**
- Shows checked and carry-on baggage allowances
- Displays weight limits when available
- Shows additional baggage fees (ancillary costs)
- Includes tooltips for user guidance
- Color-coded indicators (green for included, orange for additional fees)

### 2. `BaggageSummary` (`components/BaggageSummary.tsx`)

A compact summary component for showing essential baggage information in limited space.

**Props:**
- `transport: Transport` - The transport object to summarize

**Features:**
- Shows total included bags
- Indicates additional baggage fees with badge
- Tooltips for detailed information
- Minimal footprint for use in tight spaces

### 3. `ItineraryBaggageSummary` (`components/ItineraryBaggageSummary.tsx`)

A comprehensive summary of baggage information across an entire itinerary.

**Props:**
- `itinerary: Itinerary` - The itinerary to analyze

**Features:**
- Aggregates baggage info from all flights in the itinerary
- Shows total included bags across all flights
- Calculates total additional baggage fees
- Expandable detailed breakdown by flight
- Route and flight information for each segment

## Integration Points

### EdgeCard Component
- Uses `BaggageDisplay` for detailed baggage information in expanded view
- Uses `BaggageSummary` for compact display in flight options
- Filters baggage-related ancillary costs from general ancillaries

### ItineraryView Component
- Uses `ItineraryBaggageSummary` to show baggage overview at the top of each itinerary
- Provides passengers with quick overview of baggage allowances and costs

## Data Sources

The components read baggage information from the protobuf-generated types:

```typescript
// From Transport.flight
flight.baggagePolicy: BaggagePolicy[]
flight.ancillaryCosts: AncillaryCost[]

// BaggagePolicy structure
{
  type: BaggageType (CHECKED | CARRYON | UNSPECIFIED)
  quantity: number
  weight: number
  weightUnit: string // "KG" or "LB"
}

// AncillaryCost structure
{
  id: string
  type: string
  description: string
  cost: Cost
}
```

## Styling

Components use Chakra UI for styling with custom CSS enhancements:
- Color-coded indicators (green for included, orange for extra fees, yellow for warnings)
- Hover effects and transitions
- Responsive design
- Dark theme support

## Usage Examples

### Detailed Display
```typescript
<BaggageDisplay 
  transport={transport} 
  showTitle={true} 
  compact={false} 
/>
```

### Compact Display
```typescript
<BaggageDisplay 
  transport={transport} 
  showTitle={false} 
  compact={true} 
/>
```

### Summary Badge
```typescript
<BaggageSummary transport={transport} />
```

### Itinerary Overview
```typescript
<ItineraryBaggageSummary itinerary={itinerary} />
```

## Features

### ✅ Currently Implemented
- Display of checked and carry-on baggage allowances
- Weight limit information when available
- Additional baggage fees display
- Expandable detailed breakdowns
- Tooltips and user guidance
- Color-coded indicators
- Compact and full display modes
- Itinerary-level aggregation

### 🚀 Future Enhancements
- Baggage comparison between flight options
- Baggage preference matching
- Visual baggage capacity indicators
- Baggage policy links to airline websites
- Real-time baggage fee calculations
- Baggage packing suggestions
- Special baggage handling (sports equipment, etc.)