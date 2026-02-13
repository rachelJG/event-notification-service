# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o event-service ./cmd/api

# Final stage
FROM gcr.io/distroless/static:nonroot

WORKDIR /

COPY --from=builder /app/event-service /event-service

USER 65532:65532

ENTRYPOINT ["/event-service"]
