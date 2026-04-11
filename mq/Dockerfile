# Multi-stage Docker build for the MQ server
# Stage 1: Build the Go binary
FROM golang:1.25-alpine AS builder
WORKDIR /src

ARG ROOT=..

COPY ${ROOT}/go.mod ${ROOT}/go.sum ./
RUN go mod download
COPY . .

RUN apk add --no-cache git build-base
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/mq-server ./mq/cmd/mq-server

# Stage 2: Minimal final image (scratch)
# Contains only the compiled binary for minimal size and security
FROM scratch
COPY --from=builder /out/mq-server /mq-server
EXPOSE 8080
ENTRYPOINT ["/mq-server"]
