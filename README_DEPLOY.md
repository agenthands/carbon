# Deployment Guide (Staging)

This guide explains how to deploy the Graphiti Go server to a staging environment using Docker Compose.

## Prerequisites

- **Docker** & **Docker Compose** installed (or **Podman** & **Podman Compose**).
- **Memgraph** running (managed by `docker-compose.yml`).
- **Ollama** running locally or accessible via network (default: `http://localhost:11434`).

## Configuration

The application uses `config/config.toml` by default. You can override settings via environment variables in `.env` or `docker-compose.yml`.

### Key Environment Variables

- `PORT`: Port to listen on (default: `8080`).
- `MEMGRAPH_URI`: Memgraph connection URI (e.g., `bolt://memgraph:7687`).
- `MEMGRAPH_PASSWORD`: Database password (default: `password` in staging).
- `LLM_PROVIDER`: Set to `ollama` (default) or `openai`.
- `LLM_BASE_URL`: URL for Ollama/OpenAI API.
  - Local Ollama: `http://host.docker.internal:11434/v1` (Note: `/v1` is auto-appended if missing for Ollama provider).
- `LLM_API_KEY`: API Key (dummy for Ollama).

## Running with Docker Compose

1. **Cross-Compile Binary**:
   Run the following command on the host (outside Docker) to build the Linux executable:
   ```bash
   GOOS=linux GOARCH=arm64 go build -mod=vendor -o server-linux ./cmd/server/main.go
   # Or for x86_64: GOOS=linux GOARCH=amd64 go build -mod=vendor -o server-linux ./cmd/server/main.go
   ```

2. **Build and Start**:
   ```bash
   docker-compose up --build -d
   ```

2. **Verify Logs**:
   ```bash
   docker-compose logs -f app
   ```

3. **Check Usage Stats**:
   The application now logs token usage for each LLM request. Look for lines starting with `LLM Usage:` in the logs.
   ```bash
   docker-compose logs app | grep "LLM Usage"
   ```

4. **Verify API**:
   ```bash
   curl -X POST http://localhost:8080/search -d '{"group_id": "test", "query": "hello"}'
   ```

## Troubleshooting

- **Ollama Connection Refused**: Ensure Ollama is running on the host and bind to `0.0.0.0` (`OLLAMA_HOST=0.0.0.0 ollama serve`) or use `host.docker.internal`.
- **Memgraph Connection Failed**: Ensure Memgraph container is healthy (`docker ps`).
