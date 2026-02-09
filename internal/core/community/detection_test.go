package community

import (
	"testing"
	
	"github.com/agenthands/carbon/internal/core/model"
	"github.com/stretchr/testify/assert"
)

func TestDetect(t *testing.T) {
	nodes := []model.EntityNode{
		{UUID: "1", Name: "A"},
		{UUID: "2", Name: "B"},
		{UUID: "3", Name: "C"},
		{UUID: "4", Name: "D"},
	}

	edges := []model.EntityEdge{
		{SourceUUID: "1", TargetUUID: "2"}, // A-B
		{SourceUUID: "2", TargetUUID: "3"}, // B-C
		// D is isolated
	}

	detector := NewSimpleDetector()
	communities, err := detector.Detect(nodes, edges)

	assert.NoError(t, err)
	// Expect A-B-C as one community. D is size 1, so filtered out.
	assert.Len(t, communities, 1) 
	assert.Len(t, communities[0], 3)
	
	// Create map to verify IDs
	commMap := make(map[string]bool)
	for _, n := range communities[0] {
		commMap[n.UUID] = true
	}
	assert.True(t, commMap["1"])
	assert.True(t, commMap["2"])
	assert.True(t, commMap["3"])
}

func TestDetect_MultipleCommunities(t *testing.T) {
	nodes := []model.EntityNode{
		{UUID: "1"}, {UUID: "2"}, // C1
		{UUID: "3"}, {UUID: "4"}, // C2
	}
	
	edges := []model.EntityEdge{
		{SourceUUID: "1", TargetUUID: "2"},
		{SourceUUID: "3", TargetUUID: "4"},
	}
	
	detector := NewSimpleDetector()
	communities, err := detector.Detect(nodes, edges)
	
	assert.NoError(t, err)
	assert.Len(t, communities, 2)
}
