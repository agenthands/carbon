package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	baseURL = "http://localhost:8080"
)

func main() {
	// Wait for server to start
	time.Sleep(2 * time.Second)

	fmt.Println("Starting Integration Test...")

	groupID := "test-group-" + fmt.Sprintf("%d", time.Now().Unix())

	// 1. Ingest Messages
	fmt.Println("1. Ingesting Messages...")
	messages := []map[string]string{
		{"role": "user", "content": "My name is Alice and I am a software engineer."},
		{"role": "assistant", "content": "Nice to meet you, Alice."},
		{"role": "user", "content": "I live in San Francisco and love hiking."},
	}
	
	payload := map[string]interface{}{
		"group_id": groupID,
		"messages": messages,
	}
	
	validIngest := sendRequest("POST", "/messages", payload)
	if !validIngest {
		fmt.Println("FAILED: Ingest messages")
		os.Exit(1)
	}
	fmt.Println("PASSED: Ingest messages")

	// Allow some time for async processing if any (currently synchronous)
	time.Sleep(1 * time.Second)

	// 2. Search
	fmt.Println("2. Searching Graph...")
	searchPayload := map[string]string{
		"group_id": groupID,
		"query":    "Alice",
	}
	
	validSearch := sendRequest("POST", "/search", searchPayload)
	if !validSearch {
		fmt.Println("FAILED: Search")
		os.Exit(1)
	}
	fmt.Println("PASSED: Search")
}

func sendRequest(method, endpoint string, payload interface{}) bool {
	var body io.Reader
	if payload != nil {
		jsonBytes, _ := json.Marshal(payload)
		body = bytes.NewBuffer(jsonBytes)
	}

	req, err := http.NewRequest(method, baseURL+endpoint, body)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return false
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error sending request: %v\n", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		fmt.Printf("Request failed with status %d: %s\n", resp.StatusCode, string(respBody))
		return false
	}
	
	respBody, _ := io.ReadAll(resp.Body)
	fmt.Printf("Response: %s\n", string(respBody))

	return true
}
