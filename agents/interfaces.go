package agents

import (
	"context"

	"github.com/va6996/travelingman/pb"
)

type Planner interface {
	Plan(ctx context.Context, req PlanRequest) (*PlanResult, error)
}

type Assistant interface {
	CheckAvailability(ctx context.Context, req *pb.Itinerary) (*pb.Itinerary, error)
}

