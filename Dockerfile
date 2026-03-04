# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install required tools
RUN apk add --no-cache git gcc musl-dev

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -o app ./cmd/api

# Runtime stage
FROM alpine:latest

RUN apk add --no-cache ca-certificates postgresql-client

WORKDIR /app

# Copy the built binary from builder
COPY --from=builder /app/app .

# Copy swagger documentation
COPY --from=builder /app/docs ./docs

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=10s --timeout=5s --retries=5 \
  CMD /app/app --health || exit 1

# Run the application
CMD ["./app"]
