FROM golang:1.13 as builder

WORKDIR $GOPATH/src/kube-patch
COPY . $GOPATH/src/kube-patch
ENV GO111MODULE=on
ARG GOPROXY=https://mirrors.aliyun.com/goproxy/

RUN CGO_ENABLED=0 go build -o /root/kube-patch main.go



FROM alpine:latest
ARG TZ="Asia/Shanghai"
WORKDIR /root
COPY --from=builder /root/kube-patch /usr/local/bin
RUN apk add tzdata \
    && ln -sf /usr/share/zoneinfo/${TZ} /etc/localtime \
    && echo ${TZ} > /etc/timezone \
    && rm -rf /var/cache/apk/*
ENTRYPOINT ["kube-patch"]
