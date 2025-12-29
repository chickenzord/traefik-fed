# Build stage
FROM golang:1.25-alpine AS builder

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build arguments for version information
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown

# Build the application
RUN CGO_ENABLED=0 go build \
    -ldflags="-w -s -extldflags '-static' -X 'github.com/chickenzord/traefik-fed/internal/version.Version=${VERSION}' -X 'github.com/chickenzord/traefik-fed/internal/version.GitCommit=${GIT_COMMIT}' -X 'github.com/chickenzord/traefik-fed/internal/version.BuildDate=${BUILD_DATE}'" \
    -a -installsuffix cgo \
    -o traefik-fed ./cmd/traefik-fed

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user with UID/GID 1000
RUN addgroup -g 1000 -S appgroup && \
    adduser -u 1000 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /build/traefik-fed .

# Change ownership to non-root user
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Run the application
ENTRYPOINT ["/app/traefik-fed"]
