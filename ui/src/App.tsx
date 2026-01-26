import { useState } from 'react'
import {
    Fade,
    Spinner,
    useToast
} from '@chakra-ui/react'
import './App.css'
import { useClient } from './useClient'
import { TravelService } from './gen/protos/service_connect'
import { Itinerary } from './gen/protos/graph_pb'
import { ItineraryView } from './components/ItineraryView'

function App() {
    const [query, setQuery] = useState('')
    const [loading, setLoading] = useState(false)
    const [result, setResult] = useState<string>('')
    const [itinerary, setItinerary] = useState<Itinerary | undefined>(undefined)
    const [possibleItineraries, setPossibleItineraries] = useState<Itinerary[]>([])

    const client = useClient(TravelService)
    const toast = useToast()

    const handlePlan = async () => {
        if (!query.trim()) return

        setLoading(true)
        setResult('')
        setItinerary(undefined)
        setPossibleItineraries([])

        try {
            const response = await client.planTrip({ query })
            // setResult(response.result) // Removed
            // setItinerary(response.itinerary) // Removed
            setPossibleItineraries(response.itineraries || [])
        } catch (err) {
            toast({
                title: 'Error planning trip',
                description: err instanceof Error ? err.message : 'Unknown error',
                status: 'error',
                duration: 5000,
                isClosable: true,
            })
        } finally {
            setLoading(false)
        }
    }

    const hasContent = !!result || !!itinerary || possibleItineraries.length > 0

    return (
        <div className="app-container">
            <div className="main-container">
                {/* Header / Search Area */}
                <div className={`header-section ${hasContent ? 'top' : 'centered'}`}>
                    <div style={{ textAlign: 'center' }}>
                        <h1 className={`app-title ${hasContent ? 'small' : 'large'}`}>
                            Traveling Man
                        </h1>
                        <p className={`app-subtitle ${hasContent ? 'small' : 'large'}`}>
                            {hasContent ? "Plan another trip" : "Where do you want to go?"}
                        </p>
                    </div>

                    <div className="input-area">
                        <textarea
                            className="query-input"
                            placeholder="Ex: Trip to Paris for 5 days next month..."
                            value={query}
                            onChange={(e) => setQuery(e.target.value)}
                            onKeyDown={(e) => {
                                if (e.key === 'Enter' && !e.shiftKey) {
                                    e.preventDefault()
                                    handlePlan()
                                }
                            }}
                            disabled={loading}
                        />
                        <button
                            className="plan-button"
                            onClick={handlePlan}
                            disabled={loading}
                        >
                            Plan Trip
                        </button>
                    </div>
                </div>

                {/* Results Area */}
                {loading && (
                    <div className="loading-container">
                        <div className="loading-content">
                            <Spinner size="xl" color="green.400" thickness='4px' />
                            <span className="loading-text">Consulting travel network...</span>
                        </div>
                    </div>
                )}

                <Fade in={hasContent && !loading}>
                    <div className="results-area">
                        {/* Itinerary Visualization */}
                        {(itinerary || possibleItineraries.length > 0) && (
                            <ItineraryView itinerary={itinerary} possibleItineraries={possibleItineraries} />
                        )}
                    </div>
                </Fade>
            </div>

            {/* Meerkat Mascot */}
            <div className="meerkat-mascot">
                <img
                    src="/meerkat.png"
                    alt="Traveling Man Mascot"
                />
            </div>
        </div>
    )
}

export default App
