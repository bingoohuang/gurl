FROM golang:1.8

ADD . /go/src/github.com/bingoohuang/gurl

RUN go install github.com/bingoohuang/gurl

ENTRYPOINT ["/go/bin/bat"]