# backend/Dockerfile
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o security-copilot-server ./cmd/api

FROM alpine:3.19

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /app/security-copilot-server .
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/frontend/dist ./frontend/dist

EXPOSE 8080

CMD ["./security-copilot-server"]