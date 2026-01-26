package core

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/va6996/travelingman/pb"
)

func TestValidateItinerary_MissingNodes(t *testing.T) {
	ctx := context.Background()

	// Test JSON with missing start_loc node
	jsonData := `{
  "start_time": {
    "seconds": 1771002000
  },
  "end_time": {
    "seconds": 1771254000
  },
  "title": "Long Weekend in Paris",
  "description": "A romantic long weekend trip to Paris for two.",
  "graph": {
    "nodes": [
      {
        "id": "node_1",
        "location": "PAR"
      }
    ],
    "edges": [
      {
        "from_id": "start_loc",
        "to_id": "node_1",
        "duration_seconds": 39600
      },
      {
        "from_id": "node_1",
        "to_id": "start_loc",
        "duration_seconds": 46800
      }
    ]
  },
  "travelers": 2,
  "journey_type": 2
}`

	var itinerary pb.Itinerary
	err := json.Unmarshal([]byte(jsonData), &itinerary)
	assert.NoError(t, err)

	// This should fail validation due to missing start_loc node
	err = ValidateItinerary(ctx, &itinerary)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "FromId 'start_loc' not found in nodes")
	assert.Contains(t, err.Error(), "ToId 'start_loc' not found in nodes")
}

func TestValidateItinerary_ValidGraph(t *testing.T) {
	ctx := context.Background()

	// Test JSON with all nodes properly defined
	jsonData := `{
  "start_time": {
    "seconds": 1771002000
  },
  "end_time": {
    "seconds": 1771254000
  },
  "title": "Long Weekend in Paris",
  "description": "A romantic long weekend trip to Paris for two.",
  "graph": {
    "nodes": [
      {
        "id": "start_loc",
        "location": "SFO"
      },
      {
        "id": "node_1",
        "location": "PAR"
      }
    ],
    "edges": [
      {
        "from_id": "start_loc",
        "to_id": "node_1",
        "duration_seconds": 39600
      },
      {
        "from_id": "node_1",
        "to_id": "start_loc",
        "duration_seconds": 46800
      }
    ]
  },
  "travelers": 2,
  "journey_type": 2
}`

	var itinerary pb.Itinerary
	err := json.Unmarshal([]byte(jsonData), &itinerary)
	assert.NoError(t, err)

	// This should pass validation
	err = ValidateItinerary(ctx, &itinerary)
	assert.NoError(t, err)
}
