package server

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
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

	ollamaBaseID := os.Getenv("OLLAMA_BASE_URL")
	if ollamaBaseID == "" {
		ollamaBaseID = "http://localhost:11434"
	}

	modelName := os.Getenv("LLM_MODEL")
	if modelName == "" {
		modelName = "gpt-oss:latest"
	}

	ollamaClient, err := llm.NewOllamaClient(modelName, ollamaBaseID)
	if err != nil {
		log.Fatalf("Failed to initialize Ollama client: %v", err)
	}

	g := core.NewGraphiti(d, ollamaClient, ollamaClient) // Using Ollama for both LLM and Embedding for now

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
