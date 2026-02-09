# Runtime Stage
FROM alpine:latest

WORKDIR /app

# Install certificates for HTTPS (if needed)
RUN apk --no-cache add ca-certificates

# Copy pre-built binary from host
COPY server-linux ./server

# Copy default config (can be overridden by mount)
COPY config/config.toml ./config/config.toml

# Expose port
EXPOSE 8080

# Environment variables
ENV CONFIG_PATH=config/config.toml
ENV PORT=8080

# Run the application
CMD ["./server"]
