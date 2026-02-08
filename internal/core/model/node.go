package model

import "time"

type EntityNode struct {
	UUID          string                 `json:"uuid"`
	Name          string                 `json:"name"`
	GroupID       string                 `json:"group_id"`
	CreatedAt     time.Time              `json:"created_at"`
	Summary       string                 `json:"summary,omitempty"`
	Attributes    map[string]interface{} `json:"attributes,omitempty"`
	Labels        []string               `json:"labels"`
	NameEmbedding []float32              `json:"name_embedding,omitempty"`
}

type EpisodicNode struct {
	UUID              string    `json:"uuid"`
	Name              string    `json:"name"`
	GroupID           string    `json:"group_id"`
	CreatedAt         time.Time `json:"created_at"`
	ValidAt           time.Time `json:"valid_at"`
	Content           string    `json:"content"`
	Source            string    `json:"source"`
	SourceDescription string    `json:"source_description"`
	EntityEdges       []string  `json:"entity_edges"` // List of Edge UUIDs
}

type CommunityNode struct {
	UUID          string    `json:"uuid"`
	Name          string    `json:"name"`
	GroupID       string    `json:"group_id"`
	CreatedAt     time.Time `json:"created_at"`
	Summary       string    `json:"summary"`
	NameEmbedding []float32 `json:"name_embedding,omitempty"`
}

type SagaNode struct {
	UUID      string    `json:"uuid"`
	Name      string    `json:"name"`
	GroupID   string    `json:"group_id"`
	CreatedAt time.Time `json:"created_at"`
}
