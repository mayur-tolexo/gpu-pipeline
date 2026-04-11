# Multi-stage Docker build for the API Gateway
# Stage 1: Build the Go binary
FROM golang:1.25-alpine AS builder
WORKDIR /src

ARG ROOT=..

COPY ${ROOT}/go.mod ${ROOT}/go.sum ./
RUN go mod download
COPY . .

RUN apk add --no-cache git build-base
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api-gateway ./api-gateway/cmd/main.go

# Stage 2: Minimal final image
# Contains only the compiled binary for minimal size and security
FROM alpine:3.20
RUN apk add --no-cache ca-certificates

WORKDIR /root/

COPY --from=builder /out/api-gateway /api-gateway

EXPOSE 8000

# Health check
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
	CMD wget --no-verbose --tries=1 --spider http://localhost:8000/api/v1/health || exit 1

ENTRYPOINT ["/api-gateway"]
