FROM golang:1.25-alpine AS build
WORKDIR /src
COPY . .
RUN apk add --no-cache git gcc musl-dev
RUN go build -o /bin/collector ./cmd

FROM alpine:3.18
RUN apk add --no-cache ca-certificates
COPY --from=build /bin/collector /bin/collector
EXPOSE 8081
ENTRYPOINT ["/bin/collector"]
