# Build Stage
FROM golang:1.23.4-alpine AS builder

WORKDIR /app/controllers/session_controller

COPY controllers/session_controller ./

WORKDIR /app/session_manager

COPY session_manager/go.mod session_manager/go.sum ./

RUN go mod download

COPY session_manager .

RUN go build -o server .

# Runtime Stage
FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/session_manager/server .

EXPOSE 8080

CMD ["./server"]