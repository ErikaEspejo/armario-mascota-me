# Build stage
FROM golang:1.24 AS builder

WORKDIR /build

# Copy dependency files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o armario-mascota-me .

# Runtime stage
FROM debian:bookworm-slim

# Install Chromium, dependencies, and fonts for proper PDF/PNG rendering
RUN apt-get update && apt-get install -y \
    chromium \
    chromium-sandbox \
    ca-certificates \
    fonts-liberation \
    fonts-noto-core \
    fonts-noto-cjk \
    --no-install-recommends \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy binary from build stage
COPY --from=builder /build/armario-mascota-me .

# Copy static assets, templates, and configs
COPY --from=builder /build/static ./static
COPY --from=builder /build/templates ./templates
COPY --from=builder /build/configs ./configs

# Create cache directory for images
RUN mkdir -p cache/images

# Expose port (default 8080, but Render sets PORT env var)
EXPOSE 8080

# Run the binary
CMD ["./armario-mascota-me"]

