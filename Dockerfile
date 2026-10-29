# Stage 1: build
FROM golang:1.26.2-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /cache-app ./cmd

# Stage 2: runtime
FROM alpine:3.22
RUN apk --no-cache add ca-certificates && adduser -D -u 10001 app
WORKDIR /app
COPY --from=builder /cache-app .
USER app

EXPOSE 8080 50051
CMD ["./cache-app"]