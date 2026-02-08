package model

import "time"

type EntityEdge struct {
	UUID          string                 `json:"uuid"`
	SourceUUID    string                 `json:"source_node_uuid"`
	TargetUUID    string                 `json:"target_node_uuid"`
	GroupID       string                 `json:"group_id"`
	Name          string                 `json:"name"` // RELATES_TO
	Fact          string                 `json:"fact"`
	CreatedAt     time.Time              `json:"created_at"`
	ExpiredAt     *time.Time             `json:"expired_at,omitempty"`
	ValidAt       time.Time              `json:"valid_at"`
	InvalidAt     *time.Time             `json:"invalid_at,omitempty"`
	Episodes      []string               `json:"episodes"` // List of Episode UUIDs
	FactEmbedding []float32              `json:"fact_embedding,omitempty"`
	Attributes    map[string]interface{} `json:"attributes,omitempty"`
}

type EpisodicEdge struct {
	UUID       string    `json:"uuid"`
	SourceUUID string    `json:"source_node_uuid"` // Episode
	TargetUUID string    `json:"target_node_uuid"` // Entity
	GroupID    string    `json:"group_id"`
	CreatedAt  time.Time `json:"created_at"`
	// Relationship type is MENTIONS
}

type CommunityEdge struct {
	UUID       string    `json:"uuid"`
	SourceUUID string    `json:"source_node_uuid"` // Community
	TargetUUID string    `json:"target_node_uuid"` // Entity or Community
	GroupID    string    `json:"group_id"`
	CreatedAt  time.Time `json:"created_at"`
	// Relationship type is HAS_MEMBER
}

type HasEpisodeEdge struct {
	UUID       string    `json:"uuid"`
	SourceUUID string    `json:"source_node_uuid"` // Saga
	TargetUUID string    `json:"target_node_uuid"` // Episode
	GroupID    string    `json:"group_id"`
	CreatedAt  time.Time `json:"created_at"`
}

type NextEpisodeEdge struct {
	UUID       string    `json:"uuid"`
	SourceUUID string    `json:"source_node_uuid"` // Previous Episode
	TargetUUID string    `json:"target_node_uuid"` // Next Episode
	GroupID    string    `json:"group_id"`
	CreatedAt  time.Time `json:"created_at"`
}
