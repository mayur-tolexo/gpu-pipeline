FROM golang:1.25-alpine AS builder

WORKDIR /src

ARG ROOT=..

COPY ${ROOT}/go.mod ${ROOT}/go.sum ./
RUN go mod download
COPY . .

RUN apk add --no-cache git build-base
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/streamer ./streamer/cmd

FROM gcr.io/distroless/base-debian12

WORKDIR /
COPY --from=builder /out/streamer /streamer

ENTRYPOINT ["/streamer"]
