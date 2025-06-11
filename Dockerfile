# Multi-stage build for Go application
FROM golang:1.23-alpine AS builder

# Install git and ca-certificates (needed for private repos and HTTPS)
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/api ./cmd/api

# Final stage - minimal runtime image
FROM alpine:latest

# Install ca-certificates for HTTPS calls
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN adduser -D -s /bin/sh apiuser

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/bin/api .

# Copy email templates
COPY --from=builder /app/internal/mailer/templates ./internal/mailer/templates

# Change ownership to non-root user
RUN chown -R apiuser:apiuser /app

# Switch to non-root user
USER apiuser

# Expose port
EXPOSE 4000

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:4000/v1/api/healthcheck || exit 1

# Run the application
CMD ["./api"]
