package agents

import (
	"context"
)

// Planner is the interface for trip planning
type Planner interface {
	Plan(ctx context.Context, req PlanRequest) (*PlanResult, error)
}
