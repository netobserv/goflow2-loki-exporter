FROM golang:1.17-alpine as builder
ARG VERSION=""

RUN apk --update --no-cache add git build-base gcc

COPY . /build
WORKDIR /build

RUN go build -ldflags "-X main.version=${VERSION}" -o loki-exporter cmd/loki-exporter.go

FROM netsampler/goflow2:latest

COPY --from=builder /build/loki-exporter /

ENTRYPOINT ["./goflow2"]
