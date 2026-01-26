FROM golang:1.25-alpine AS builder

# Build arguments
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG VERSION_TYPE=dev

# Install build dependencies
RUN apk add --no-cache make git

WORKDIR /app

# Copy source code
COPY . .

# Build using Makefile with version args
RUN VERSION=${VERSION} GIT_COMMIT=${GIT_COMMIT} VERSION_TYPE=${VERSION_TYPE} make build

# Final stage
FROM alpine:latest

# Create directories
RUN mkdir -p /etc/projecthorizon/config /var/lib/projecthorizon/www

# Copy binary from builder
COPY --from=builder /app/build/horizon /usr/bin/horizon

# Copy public files
COPY --from=builder /app/public /var/lib/projecthorizon/www

# Copy config
COPY --from=builder /app/config /etc/projecthorizon/config

# Expose port
EXPOSE 8080

# Set entrypoint
CMD ["/usr/bin/horizon"]
