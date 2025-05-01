# Build Stage
FROM golang:1.23.4-alpine AS builder

WORKDIR /app/controllers/session

COPY controllers/session ./

WORKDIR /app/session-manager

COPY session-manager/go.mod session-manager/go.sum ./

RUN go mod download

COPY session-manager .

RUN go build -o server .

# Runtime Stage
FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/session-manager/server .

EXPOSE 8080

CMD ["./server"]