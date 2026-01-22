FROM golang:alpine AS builder

# Install build dependencies
RUN apk add --no-cache make git bash

# Mock sudo and systemctl to success (exit 0) so make install works
RUN echo '#!/bin/sh' > /usr/bin/sudo && \
    chmod +x /usr/bin/sudo && \
    echo 'exec "$@"' > /usr/bin/sudo && \
    echo '#!/bin/sh' > /usr/bin/systemctl && \
    chmod +x /usr/bin/systemctl

WORKDIR /app

# Copy source code
COPY . .

# Build using Makefile
RUN make build

# Install using Makefile (mocking system behavior)
# Overriding INSTALL_PREFIX to /usr so binary goes to /usr/bin/horizon
RUN make install INSTALL_PREFIX=/usr

# Final stage
FROM alpine:latest

# Copy installed artifacts from builder
COPY --from=builder /usr/bin/horizon /usr/bin/horizon
COPY --from=builder /etc/projecthorizon /etc/projecthorizon
COPY --from=builder /var/lib/projecthorizon /var/lib/projecthorizon

# Expose port (assuming 8080 based on typical Go apps, but strictly following user request for now)
# (User didn't ask to expose port, but good practice. I'll leave it out to be strict unless seen in code)

# Set entrypoint
CMD ["/usr/bin/horizon"]
