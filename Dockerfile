ARG GO_VERSION=latest 

FROM golang:${GO_VERSION} AS builder

ENV GOPROXY https://goproxy.cn,direct

ENV CONFIG="conf.toml"

WORKDIR $GOPATH/src

ADD . $GOPATH/src

RUN go build -o amqtt

EXPOSE 1884 2884 8086

ENTRYPOINT ["./amqtt", "-c", $CONFIG]