# STAGE 1: Build the Go binary
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum first to take advantage of Docker layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the backend source code
COPY . .

# Compile a static Linux binary named 'server'
RUN CGO_ENABLED=0 GOOS=linux go build -o server .

# STAGE 2: Ultra-lightweight runner
FROM alpine:latest

WORKDIR /app

# Copy only the compiled binary from Stage 1
COPY --from=builder /app/server .
# Copy .env file so Go can read MongoDB URI and secrets
COPY .env .

EXPOSE 8000

CMD ["./server"]
