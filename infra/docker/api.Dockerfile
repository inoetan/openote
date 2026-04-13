# Stage 1: build
FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /server ./cmd/server

# Stage 2: runtime
FROM alpine:3.19

COPY --from=builder /server /server

EXPOSE 8080

CMD ["/server"]
