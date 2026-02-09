package community

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/agenthands/carbon/internal/core/model"
)

func TestLPA_DisconnectedComponents(t *testing.T) {
	// Graph: [1-2-3-1] (Triangle A) ... [4-5-6-4] (Triangle B)
	// Completely disconnected
	nodes := []model.EntityNode{
		{UUID: "1"}, {UUID: "2"}, {UUID: "3"},
		{UUID: "4"}, {UUID: "5"}, {UUID: "6"},
	}
	edges := []model.EntityEdge{
		{SourceUUID: "1", TargetUUID: "2"}, {SourceUUID: "2", TargetUUID: "3"}, {SourceUUID: "3", TargetUUID: "1"},
		{SourceUUID: "4", TargetUUID: "5"}, {SourceUUID: "5", TargetUUID: "6"}, {SourceUUID: "6", TargetUUID: "4"},
	}

	detector := NewLabelPropagationDetector()
	communities, err := detector.Detect(nodes, edges)
	assert.NoError(t, err)

	assert.Len(t, communities, 2)
	// Sort for stability in assertion
	sort.Slice(communities, func(i, j int) bool {
		return len(communities[i]) > len(communities[j]) // Both size 3
	})
	
	// Check contents (approximate)
	count3 := 0
	for _, c := range communities {
		if len(c) == 3 {
			count3++
		}
	}
	assert.Equal(t, 2, count3)
}

func TestLPA_BridgeNode(t *testing.T) {
	// Graph: [1-2-3-1] --(3-4)-- [4-5-6-4]
	// Two triangles connected by edge 3-4. 
	// LPA typically keeps them separate as intra-cluster edges > inter-cluster edge.
	
	nodes := []model.EntityNode{
		{UUID: "1"}, {UUID: "2"}, {UUID: "3"},
		{UUID: "4"}, {UUID: "5"}, {UUID: "6"},
	}
	edges := []model.EntityEdge{
		{SourceUUID: "1", TargetUUID: "2"}, {SourceUUID: "2", TargetUUID: "3"}, {SourceUUID: "3", TargetUUID: "1"},
		// Bridge
		{SourceUUID: "3", TargetUUID: "4"},
		// Triangle 2
		{SourceUUID: "4", TargetUUID: "5"}, {SourceUUID: "5", TargetUUID: "6"}, {SourceUUID: "6", TargetUUID: "4"},
	}

	detector := NewLabelPropagationDetector()
	communities, err := detector.Detect(nodes, edges)
	assert.NoError(t, err)

	// Could be 1 or 2 depending on propagation.
	// With unweighted LPA and tie-breaking by max label, 3 and 4 have 2 strong neighbors vs 1 bridge neighbor.
	// So 3 should stick to {1,2} and 4 to {5,6}.
	// Result: 2 communities.
	assert.Len(t, communities, 2)
}

func TestLPA_LargeClique(t *testing.T) {
	// Clique of 5 nodes
	nodes := []model.EntityNode{
		{UUID: "1"}, {UUID: "2"}, {UUID: "3"}, {UUID: "4"}, {UUID: "5"},
	}
	var edges []model.EntityEdge
	// Fully connected
	for i := range nodes {
		for j := i + 1; j < len(nodes); j++ {
			edges = append(edges, model.EntityEdge{
				SourceUUID: nodes[i].UUID,
				TargetUUID: nodes[j].UUID,
			})
		}
	}

	detector := NewLabelPropagationDetector()
	communities, err := detector.Detect(nodes, edges)
	assert.NoError(t, err)

	assert.Len(t, communities, 1)
	assert.Len(t, communities[0], 5)
}
