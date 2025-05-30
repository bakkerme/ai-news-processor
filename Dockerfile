# Build stage
FROM golang:1.24-alpine3.20 AS builder

WORKDIR /app

# Copy the source code
COPY . .

# Build the application
RUN GOOS=linux go build -o /app/main ./internal

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/main /app/main
COPY --from=builder /app/personas /app/personas

COPY build/crontab /etc/cron.d/appcron
COPY build/init.sh /app/init.sh

# Set execute permissions
RUN chmod +x /app/main /app/init.sh

# Command to run the executable
CMD ["sh", "/app/init.sh"]
