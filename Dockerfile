FROM golang:1.22 AS builder

ENV DEBIAN_FRONTEND=noninteractive
RUN apt update && apt install -y protobuf-compiler && \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28 && \
 go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
ADD . /go/src/grpcprobe
WORKDIR /go/src/grpcprobe
RUN go generate && ls -R && sleep 5
RUN CGO_ENABLED=0 go build -o / ./cmd/grpcprober

FROM scratch

COPY --from=builder /grpcprober /

EXPOSE 12345
EXPOSE 9090

ENTRYPOINT ["/grpcprober"]
