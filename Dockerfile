FROM golang:1.15 as builder
WORKDIR /go/src/github.com/l1b0k/aliyun-ddns/
COPY . .
RUN CGO_ENABLED=0 go build \
    -ldflags "-X \"github.com/l1b0k/aliyun-ddns/version.gitVer=`git rev-parse --short HEAD 2>/dev/null`\" -X \"github.com/l1b0k/aliyun-ddns/version.buildTime=$(date +%F_%H:%M:%S)\" -X \"github.com/l1b0k/aliyun-ddns/version.version=v0.0.1\" " \
    -o ddns .

FROM ubuntu:20.04
LABEL maintainer="4043362+l1b0k@users.noreply.github.com"
WORKDIR /usr/bin
COPY --from=builder /go/src/github.com/l1b0k/aliyun-ddns/ddns /usr/bin/ddns
ENTRYPOINT ["/usr/bin/ddns"]
