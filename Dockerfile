# Build stage
FROM golang:1.24-bookworm AS builder

WORKDIR /app

# Copy the source code
COPY . .

# Build the application
RUN GOOS=linux go build -o /app/main ./internal

# Final stage
FROM debian:bookworm-slim

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/main /app/main

COPY build/crontab /etc/cron.d/appcron
COPY build/init.sh /app/init.sh

# Install CA certificates (needed for HTTPS requests)
RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates cron && \
    rm -rf /var/lib/apt/lists/*

# Command to run the executable
CMD ["sh", "/app/init.sh"]
