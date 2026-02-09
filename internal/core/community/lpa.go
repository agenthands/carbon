package community

import (
	"sort"

	"github.com/agenthands/carbon/internal/core/model"
)

// LabelPropagationDetector implements community detection using Label Propagation Algorithm (LPA).
type LabelPropagationDetector struct {
	MaxIterations int
}

func NewLabelPropagationDetector() *LabelPropagationDetector {
	return &LabelPropagationDetector{
		MaxIterations: 20, // Default max iterations
	}
}

func (d *LabelPropagationDetector) Detect(nodes []model.EntityNode, edges []model.EntityEdge) ([][]model.EntityNode, error) {
	if len(nodes) == 0 {
		return nil, nil
	}

	// 1. Initialize Adjacency List (Graph)
	// Weighted by edge count between nodes (simple weighting for now: 1 edge = weight 1)
	// Or use edge attributes/frequency if available. 
	// For now, simple undirected unweighted graph, but we count multiple edges as stronger connection.
	
	adj := make(map[string]map[string]int) // node -> neighbor -> weight
	nodeMap := make(map[string]model.EntityNode)
	
	for _, n := range nodes {
		nodeMap[n.UUID] = n
		adj[n.UUID] = make(map[string]int)
	}

	for _, e := range edges {
		if _, ok := nodeMap[e.SourceUUID]; !ok { continue }
		if _, ok := nodeMap[e.TargetUUID]; !ok { continue }
		
		adj[e.SourceUUID][e.TargetUUID]++
		adj[e.TargetUUID][e.SourceUUID]++ // Undirected
	}

	// 2. Initialize Labels
	// Each node starts with its own unique label (its UUID)
	labels := make(map[string]string)
	for _, n := range nodes {
		labels[n.UUID] = n.UUID
	}
	
	// 3. Propagation Loop
	// To prevent oscillations and bias, we should ideally shuffle node processing order each iteration.
	// For simplicity, we iterate over map (which is semi-random in Go) or better, slice.
	nodeUUIDs := make([]string, len(nodes))
	for i, n := range nodes {
		nodeUUIDs[i] = n.UUID
	}

	for iter := 0; iter < d.MaxIterations; iter++ {
		changeCount := 0
		
		// In a real implementation we might shuffle nodeUUIDs here.
		
		for _, u := range nodeUUIDs {
			neighbors := adj[u]
			if len(neighbors) == 0 {
				continue
			}

			// Count label frequencies among neighbors weighted by edge weight
			labelCounts := make(map[string]int)
			maxCount := 0
			
			for v, weight := range neighbors {
				label := labels[v]
				labelCounts[label] += weight
				if labelCounts[label] > maxCount {
					maxCount = labelCounts[label]
				}
			}
			
			// Find all labels with max frequency (to break ties)
			var candidates []string
			for label, count := range labelCounts {
				if count == maxCount {
					candidates = append(candidates, label)
				}
			}
			
			// Tie-breaking: 
			// 1. If current label is max, keep it
			// 2. Else pick lexicographically largest (deterministic) or random
			// We'll use lexicographically largest for stability
			sort.Strings(candidates)
			bestLabel := candidates[len(candidates)-1]
			
			if labels[u] != bestLabel {
				labels[u] = bestLabel
				changeCount++
			}
		}

		if changeCount == 0 {
			break
		}
	}
	
	// 4. Group by Label
	clusters := make(map[string][]model.EntityNode)
	for uuid, label := range labels {
		if node, ok := nodeMap[uuid]; ok {
			clusters[label] = append(clusters[label], node)
		}
	}
	
	var communities [][]model.EntityNode
	for _, cluster := range clusters {
		if len(cluster) >= 2 { // Filter singletons per spec/discussion
			communities = append(communities, cluster)
		}
	}
	
	return communities, nil
}
