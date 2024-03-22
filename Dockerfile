FROM golang:1.22

ADD . /go/src/grpcprobe
WORKDIR /go/src/grpcprobe
RUN go build -o / ./cmd/grpcprober

EXPOSE 12345

ENTRYPOINT ["/grpcprober"]
