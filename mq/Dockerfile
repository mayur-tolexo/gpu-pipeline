# Build stage
FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod .
COPY . .
RUN apk add --no-cache git build-base
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/mq-server ./cmd/mq-server

# Final image
FROM scratch
COPY --from=builder /out/mq-server /mq-server
EXPOSE 8080
ENTRYPOINT ["/mq-server"]
