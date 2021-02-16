FROM golang:1.15 as builder
WORKDIR /go/src/github.com/l1b0k/aliyun-ddns/
COPY . .
RUN CGO_ENABLED=0 go build \
    -ldflags "-X \"github.com/l1b0k/aliyun-ddns/version.gitVer=`git rev-parse --short HEAD 2>/dev/null`\" -X \"github.com/l1b0k/aliyun-ddns/version.buildTime=$(date +%F_%H:%M:%S)\" -X \"github.com/l1b0k/aliyun-ddns/version.version=$(git tag)\" " \
    -o ddns .

FROM debian:stretch-slim
LABEL maintainer="4043362+l1b0k@users.noreply.github.com"
WORKDIR /usr/bin
RUN apt-get update && apt-get install -y ca-certificates && apt-get clean -y && rm -rf /var/lib/apt/lists/*
COPY --from=builder /go/src/github.com/l1b0k/aliyun-ddns/ddns /usr/bin/ddns
ENTRYPOINT ["/usr/bin/ddns"]
