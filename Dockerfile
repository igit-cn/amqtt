ARG GO_VERSION=latest

FROM golang:${GO_VERSION} AS builder

ENV GOPROXY https://goproxy.cn,direct

ENV CONFIG="conf.toml"

WORKDIR /go/src

ADD . /go/src

RUN go build -o /go/amqtt

EXPOSE 1884 2884 8086

ENTRYPOINT [ "/bin/bash", "-c", "/go/amqtt -c ${CONFIG}" ]