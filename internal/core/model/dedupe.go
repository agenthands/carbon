package model

type DuplicatePair struct {
	OriginalUUID string `json:"original_uuid"` // The existing node UUID
	DuplicateUUID string `json:"duplicate_uuid"` // The new node UUID (or temporary ID)
	Confidence   float64 `json:"confidence"`
}

type DeduplicationResult struct {
	Duplicates []DuplicatePair `json:"duplicates"`
}

type ContradictionResult struct {
	ContradictedEdgeUUIDs []string `json:"contradicted_edge_uuids"`
}
