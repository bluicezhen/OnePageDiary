FROM golang:1.24 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN go build -o /app/onepage-server ./cmd/server

FROM debian:bookworm-slim
WORKDIR /app
COPY --from=builder /app/onepage-server /app/onepage-server
EXPOSE 8080
CMD ["/app/onepage-server"]
