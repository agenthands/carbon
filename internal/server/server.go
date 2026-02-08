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
	// 1. Load Config
	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "config/config.toml"
	}
	
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Printf("Warning: Could not load config/config.toml: %v. Using empty config", err)
		// Try fallback if really needed, but better to fail or use defaults
		cfg = &config.Config{}
	}

	// 2. Override Secrets with Env Vars (ONLY Secrets)
	if envAPIKey := os.Getenv("LLM_API_KEY"); envAPIKey != "" {
		cfg.LLM.APIKey = envAPIKey
	}
	// Password for DB (Secret)
	if envDBPass := os.Getenv("MEMGRAPH_PASSWORD"); envDBPass != "" {
		cfg.Memgraph.Password = envDBPass
	}
	
	// 3. Initialize Memgraph Driver
	// Use config URI/User, default if missing
	if cfg.Memgraph.URI == "" {
		cfg.Memgraph.URI = "bolt://localhost:7687"
	}
	
	d, err := driver.NewMemgraphDriver(cfg.Memgraph.URI, cfg.Memgraph.User, cfg.Memgraph.Password)
	if err != nil {
		log.Fatalf("Failed to connect to Memgraph: %v", err)
	}

	// 4. Default LLM if missing
	if cfg.LLM.Provider == "" {
		cfg.LLM.Provider = "ollama"
		cfg.LLM.Model = "gpt-oss:latest"
		cfg.LLM.BaseURL = "http://localhost:11434"
	}

	// 5. Initialize LLM Client via Factory
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
