# Build Stage
FROM golang:1.23.4-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o server .

# Runtime Stage
FROM alpine:latest

WORKDIR /root/

COPY ./data.json .

COPY --from=builder /app/server .

EXPOSE 8080

CMD ["./server"]