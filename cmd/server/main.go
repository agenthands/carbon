package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/agenthands/carbon/internal/server"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using defaults")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := server.NewServer()
	r := srv.SetupRouter()

	log.Printf("Starting server on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
