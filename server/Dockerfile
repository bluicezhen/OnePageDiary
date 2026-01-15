FROM golang:1.24-alpine AS builder

WORKDIR /app
RUN apk add --no-cache build-base
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=1 go build -o /app/onepage-server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/onepage-server /app/onepage-server
EXPOSE 8080
CMD ["/app/onepage-server"]
