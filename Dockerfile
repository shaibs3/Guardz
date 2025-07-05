# Build stage
FROM golang:1.24.2-alpine AS builder

RUN apk add --no-cache make

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN make build

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/bin/guardz .
# Expose the application port
EXPOSE 8080

# Run the application
CMD ["/app/guardz"]
