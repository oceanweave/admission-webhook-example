FROM golang:1.17 as builder

RUN apt-get -y update && apt-get -y install upx

WORKDIR /workspace
COPY go.mod go.mod
COPY go.sum go.sum

#COPY main.go main.go
COPY webhook/ webhook/
COPY tls/ tls/
COPY pkg/ pkg/

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64
ENV GO111MODULE=on
ENV GOPROXY="https://goproxy.cn"
#ENV GIT_TERMINAL_PROMPT=1

RUN go mod download && \
#    go get github.com/oceanweave/admission-webhook-sample/pkg && \
    go build -a -o admission-registry ./webhook && \
    go build -a -o tls-manager ./tls && \
    upx admission-registry tls-manager
# upx 是压缩

FROM alpine:3.9.2 as manager
COPY --from=builder /workspace/admission-registry .
ENTRYPOINT ["/admission-registry"]
# docker build --target manager -t dfy007/admission-webhook-example:v1.9 -f Dockerfile .

FROM alpine:3.9.2 as tls
COPY --from=builder /workspace/tls-manager .
ENTRYPOINT ["/tls-manager"]
# docker build --target tls -t dfy007/admission-registry-tls:v1.9 -f Dockerfile .