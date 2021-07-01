ARG GO_VERSION=latest 

FROM golang:${GO_VERSION} AS builder

ENV GOPROXY https://goproxy.cn,direct

WORKDIR $GOPATH/src

ADD . $GOPATH/src

RUN go build -o amqtt

EXPOSE 1884 2884 8086

ENTRYPOINT ["./amqtt"]