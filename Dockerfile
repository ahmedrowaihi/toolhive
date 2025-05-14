# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build both binaries
RUN CGO_ENABLED=0 GOOS=linux go build -o thv-web ./cmd/thv-web
RUN CGO_ENABLED=0 GOOS=linux go build -o thv ./cmd/thv

# Final stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Copy the binaries from builder
COPY --from=builder /app/thv-web .
COPY --from=builder /app/thv /usr/local/bin/

# Copy static web files
COPY --from=builder /app/pkg/gui/web/static ./pkg/gui/web/static

# Set environment variables
ENV PORT=8080
ENV TOOLHIVE_AUTH_TOKEN=""

# Expose the port
EXPOSE 8080

# Run the application
CMD ["./thv-web"] 