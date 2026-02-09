package config

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
)

type ExtractionPrompts struct {
	Nodes string `toml:"nodes"`
	Edges string `toml:"edges"`
}

type DeduplicationPrompts struct {
	Nodes string `toml:"nodes"`
	Edges string `toml:"edges"`
}

type SummaryPrompts struct {
	Nodes         string `toml:"nodes"`
	Communities   string `toml:"communities"`
	CommunityName string `toml:"community_name"`
}

type LLMConfig struct {
	Provider       string `toml:"provider"`
	Model          string `toml:"model"`
	EmbeddingModel string `toml:"embedding_model"`
	APIKey         string `toml:"api_key"`
	BaseURL        string `toml:"base_url"`
}

type MemgraphConfig struct {
	URI      string `toml:"uri"`
	User     string `toml:"user"`
	Password string `toml:"password"`
}

type ConcurrencyConfig struct {
	BulkIngest int `toml:"bulk_ingest"`
	BulkSearch int `toml:"bulk_search"`
}

type Config struct {
	LLM           LLMConfig            `toml:"llm"`
	Memgraph      MemgraphConfig       `toml:"memgraph"`
	Extraction    ExtractionPrompts    `toml:"extraction"`
	Deduplication DeduplicationPrompts `toml:"deduplication"`
	Summary       SummaryPrompts       `toml:"summary"`
	Concurrency   ConcurrencyConfig    `toml:"concurrency"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file '%s': %w", path, err)
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse TOML: %w", err)
	}

	return &cfg, nil
}
