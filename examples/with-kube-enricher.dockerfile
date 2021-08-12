FROM golang:alpine as builder
ARG VERSION=""

RUN apk --update --no-cache add git build-base gcc

COPY src /build
WORKDIR /build

RUN go build -ldflags "-X main.version=${VERSION}" -o loki-exporter

FROM quay.io/jotak/goflow2:kube-latest

COPY --from=builder /build/loki-exporter /

ENTRYPOINT ["./goflow2"]
