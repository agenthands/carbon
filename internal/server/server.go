package server

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/agenthands/carbon/internal/config"
	"github.com/agenthands/carbon/internal/core"
	"github.com/agenthands/carbon/internal/driver"
	"github.com/agenthands/carbon/internal/llm"
)

type Server struct {
	Graphiti *core.Graphiti
}

func NewServer() *Server {
	// Initialize components
	dbURI := os.Getenv("MEMGRAPH_URI")
	if dbURI == "" {
		dbURI = "bolt://localhost:7687"
	}
	dbUser := os.Getenv("MEMGRAPH_USER")
	dbPass := os.Getenv("MEMGRAPH_PASSWORD")

	d, err := driver.NewMemgraphDriver(dbURI, dbUser, dbPass)
	if err != nil {
		log.Fatalf("Failed to connect to Memgraph: %v", err)
	}

	// Load Config
	cfg, err := config.Load("config/config.toml")
	if err != nil {
		log.Printf("Warning: Could not load config/config.toml: %v. Using empty config (This will panic if code expects config)", err)
		// For robustness, maybe we should have default config fallback, but for now let's fail or assume file exists
		// Actually, let's look for env var for config path
		cfgPath := os.Getenv("CONFIG_PATH")
		if cfgPath == "" {
			cfgPath = "config/config.toml"
		}
		
		cfg, err = config.Load(cfgPath)
		if err != nil {
			log.Fatalf("Failed to load configuration: %v", err)
		}
	}

	// Override config with env vars if present (simple override logic)
	if envProvider := os.Getenv("LLM_PROVIDER"); envProvider != "" {
		cfg.LLM.Provider = envProvider
	}
	if envModel := os.Getenv("LLM_MODEL"); envModel != "" {
		cfg.LLM.Model = envModel
	}
	if envEmbeddingModel := os.Getenv("LLM_EMBEDDING_MODEL"); envEmbeddingModel != "" {
		cfg.LLM.EmbeddingModel = envEmbeddingModel
	}
	if envAPIKey := os.Getenv("LLM_API_KEY"); envAPIKey != "" {
		cfg.LLM.APIKey = envAPIKey
	}
	if envBaseURL := os.Getenv("LLM_BASE_URL") ; envBaseURL != "" {
		cfg.LLM.BaseURL = envBaseURL
	}
	
	// Default to Ollama if provider is empty
	if cfg.LLM.Provider == "" {
		cfg.LLM.Provider = "ollama"
		cfg.LLM.Model = "gpt-oss:latest"
		cfg.LLM.BaseURL = "http://localhost:11434"
	}

	// Initialize LLM Client via Factory
	llmClient, embedderClient, err := llm.NewClient(context.Background(), cfg.LLM)
	if err != nil {
		log.Fatalf("Failed to initialize LLM client: %v", err)
	}

	g := core.NewGraphiti(d, llmClient, embedderClient, cfg)

	return &Server{
		Graphiti: g,
	}
}

func (s *Server) SetupRouter() *gin.Engine {
	r := gin.Default()

	r.POST("/messages", s.AddMessages)
	r.POST("/search", s.Search)

	return r
}

type AddMessageRequest struct {
	GroupID  string `json:"group_id"`
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
}

func (s *Server) AddMessages(c *gin.Context) {
	var req AddMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	for _, msg := range req.Messages {
		err := s.Graphiti.AddEpisode(c.Request.Context(), req.GroupID, "message", msg.Content)
		if err != nil {
			log.Printf("Failed to add episode: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process message"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

type SearchRequest struct {
	GroupID string `json:"group_id"`
	Query   string `json:"query"`
}

func (s *Server) Search(c *gin.Context) {
	var req SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	results, err := s.Graphiti.Search(c.Request.Context(), req.GroupID, req.Query)
	if err != nil {
		log.Printf("Failed to search: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}
