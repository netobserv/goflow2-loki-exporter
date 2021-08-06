FROM golang:alpine as builder
ARG LDFLAGS=""

RUN apk --update --no-cache add git build-base gcc

COPY src /build
WORKDIR /build

RUN go build -o loki-exporter

FROM quay.io/jotak/goflow:v2-kube

COPY --from=builder /build/loki-exporter /

ENTRYPOINT ["./goflow2"]