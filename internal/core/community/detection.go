package community

import (
	"github.com/agenthands/carbon/internal/core/model"
)

type CommunityDetector interface {
	Detect(nodes []model.EntityNode, edges []model.EntityEdge) ([][]model.EntityNode, error)
}

type SimpleDetector struct {}

func NewSimpleDetector() CommunityDetector {
	return NewLabelPropagationDetector()
}

func (d *SimpleDetector) Detect(nodes []model.EntityNode, edges []model.EntityEdge) ([][]model.EntityNode, error) {
	nodeMap := make(map[string]model.EntityNode)
	adj := make(map[string][]string)
	
	for _, n := range nodes {
		nodeMap[n.UUID] = n
	}

	for _, e := range edges {
		// Undirected graph for community detection based on simple connectivity
		// Only consider edges where both nodes are in the provided node list (which should be all nodes in group)
		if _, ok := nodeMap[e.SourceUUID]; !ok {
			continue
		}
		if _, ok := nodeMap[e.TargetUUID]; !ok {
			continue
		}

		adj[e.SourceUUID] = append(adj[e.SourceUUID], e.TargetUUID)
		adj[e.TargetUUID] = append(adj[e.TargetUUID], e.SourceUUID)
	}

	visited := make(map[string]bool)
	var communities [][]model.EntityNode

	for _, n := range nodes {
		if !visited[n.UUID] {
			componentUUIDs := []string{}
			d.dfs(n.UUID, adj, visited, &componentUUIDs)
			
			// Only consider components with more than 1 node as a "community"? 
			// Or even single nodes?
			// Roadmap objectives: "Generate summary insights for clusters of related entities."
			// Single nodes are not clusters. Let's filter for size > 1 for now, or keep all.
			// Keeping all allows summarizing isolated entities which might be useful, 
			// but "Community" implies > 1. 
			// Let's keep all for completeness, but typically community detection ignores singletons.
			// Let's keep size >= 2.
			if len(componentUUIDs) >= 2 {
				var community []model.EntityNode
				for _, uuid := range componentUUIDs {
					if node, exists := nodeMap[uuid]; exists {
						community = append(community, node)
					}
				}
				communities = append(communities, community)
			}
		}
	}

	return communities, nil
}

func (d *SimpleDetector) dfs(u string, adj map[string][]string, visited map[string]bool, component *[]string) {
	visited[u] = true
	*component = append(*component, u)
	for _, v := range adj[u] {
		if !visited[v] {
			d.dfs(v, adj, visited, component)
		}
	}
}
