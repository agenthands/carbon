package model

// Matches Python ExtractedEntity in graphiti_core/prompts/extract_nodes.py
type ExtractedEntity struct {
	Name         string                 `json:"name"`
	EntityTypeID int                    `json:"entity_type_id"`
	Attributes   map[string]interface{} `json:"attributes,omitempty"`
}

// Matches Python ExtractedEntities
type ExtractedEntities struct {
	ExtractedEntities []ExtractedEntity `json:"extracted_entities"`
}

// Matches Python EntitySummary
type EntitySummary struct {
	Summary string `json:"summary"`
}

type CommunityName struct {
	Name string `json:"name"`
}

// Prompt context data structure
type ExtractionContext struct {
	EntityTypes         string   `json:"entity_types"`
	PreviousEpisodes    []string `json:"previous_episodes"`
	EpisodeContent      string   `json:"episode_content"`
	CustomInstructions  string   `json:"custom_extraction_instructions,omitempty"`
	SourceDescription   string   `json:"source_description,omitempty"`
}

// Matches Python ExtractedEdge in graphiti_core/prompts/extract_edges.py
type ExtractedEdge struct {
	SourceNodeUUID string `json:"source_node_uuid"`
	TargetNodeUUID string `json:"target_node_uuid"`
	RelationType   string `json:"relation_type"`
	Fact           string `json:"fact"`
}

type ExtractedEdges struct {
	ExtractedEdges []ExtractedEdge `json:"extracted_edges"`
}
