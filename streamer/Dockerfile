FROM golang:1.22 AS builder

WORKDIR /app
COPY . .

RUN go mod tidy
RUN go build -o streamer cmd/main.go

FROM gcr.io/distroless/base-debian12

WORKDIR /
COPY --from=builder /app/streamer /streamer

# CSV mounted via volume
VOLUME ["/data"]

ENTRYPOINT ["/streamer"]