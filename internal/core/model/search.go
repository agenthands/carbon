package model

type SearchResult struct {
	UUID       string    `json:"uuid"`
	Name       string    `json:"name"`
	Summary    string    `json:"summary"`
	Score      float64   `json:"score"`      // Combined Hybrid Score
	VectorScore float64  `json:"vector_score,omitempty"`
	TextScore   float64  `json:"text_score,omitempty"`
}
