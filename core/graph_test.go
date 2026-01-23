package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/va6996/travelingman/pb"
)

func TestHasCycle(t *testing.T) {
	tests := []struct {
		name     string
		graph    *pb.Graph
		expected bool
	}{
		{
			name:     "Empty graph",
			graph:    &pb.Graph{},
			expected: false,
		},
		{
			name: "Simple no cycle",
			graph: &pb.Graph{
				Nodes: []*pb.Node{{Id: "A"}, {Id: "B"}},
				Edges: []*pb.Edge{{FromId: "A", ToId: "B"}},
			},
			expected: false,
		},
		{
			name: "Simple cycle",
			graph: &pb.Graph{
				Nodes: []*pb.Node{{Id: "A"}, {Id: "B"}},
				Edges: []*pb.Edge{
					{FromId: "A", ToId: "B"},
					{FromId: "B", ToId: "A"},
				},
			},
			expected: true,
		},
		{
			name: "Complex cycle",
			graph: &pb.Graph{
				Nodes: []*pb.Node{{Id: "A"}, {Id: "B"}, {Id: "C"}},
				Edges: []*pb.Edge{
					{FromId: "A", ToId: "B"},
					{FromId: "B", ToId: "C"},
					{FromId: "C", ToId: "A"},
				},
			},
			expected: true,
		},
		{
			name: "Disconnected cycle",
			graph: &pb.Graph{
				Nodes: []*pb.Node{{Id: "A"}, {Id: "B"}, {Id: "C"}, {Id: "D"}},
				Edges: []*pb.Edge{
					{FromId: "A", ToId: "B"},
					{FromId: "C", ToId: "D"},
					{FromId: "D", ToId: "C"},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, HasCycle(tt.graph))
		})
	}
}
