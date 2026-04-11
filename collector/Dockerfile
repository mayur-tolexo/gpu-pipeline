FROM golang:1.25-alpine AS build
WORKDIR /src

ARG ROOT=..

COPY ${ROOT}/go.mod ${ROOT}/go.sum ./
RUN go mod download
COPY . .

RUN apk add --no-cache git gcc musl-dev
RUN CGO_ENABLED=1 GOOS=linux CGO_CFLAGS="-O -D_LARGEFILE64_SOURCE" go build -o /bin/collector ./collector/cmd

FROM alpine:3.18
RUN apk add --no-cache ca-certificates
COPY --from=build /bin/collector /bin/collector
EXPOSE 8081
ENTRYPOINT ["/bin/collector"]
