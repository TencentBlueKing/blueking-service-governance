# Example (run from blueking-service-governance root):
#    docker build -f bkms-server/Dockerfile \
#       -t bkms/bkms-server:0.0.1-alpha.1 \
#       --build-arg VERSION="$(git describe --always)" \
#       --build-arg GIT_HASH="$(git rev-parse HEAD)" \
#       --build-arg GOPROXY="https://goproxy.cn,direct" \
#       --build-arg GOSUMDB="<...>" .
#
# 注意：不传 --build-arg VERSION/GIT_HASH 时，/version 接口的 version、gitHash 字段将为空
#
# -------------- builder container --------------
FROM golang:1.25.7-alpine3.22 AS builder

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.tencent.com/g' /etc/apk/repositories

RUN apk --update --no-cache add make git

WORKDIR /go/src/

ARG VERSION
ARG GIT_HASH
ARG GOPROXY
ARG GOSUMDB

ENV GO111MODULE=on
ENV GOFLAGS=-mod=readonly

COPY bkms-server/go.mod go.mod
COPY bkms-server/go.sum go.sum
COPY libs/bkms-adapter/go.mod /go/libs/bkms-adapter/go.mod
COPY libs/bkms-adapter/go.sum /go/libs/bkms-adapter/go.sum

RUN printf 'go 1.25.7\n\nuse (\n\t./src\n\t./libs/bkms-adapter\n)\n' > /go/go.work

RUN go mod download \
    && cd /go/libs/bkms-adapter \
    && go mod download

COPY bkms-server/ .
COPY libs/bkms-adapter /go/libs/bkms-adapter

RUN make build VERSION=$VERSION GIT_HASH=$GIT_HASH

# -------------- runner container --------------
FROM alpine:3.22 AS runner

ENV TZ=Asia/Shanghai

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.tencent.com/g' /etc/apk/repositories

RUN apk --update --no-cache add bash curl tzdata

WORKDIR /app

COPY --from=builder /go/src/build/bkms-server /usr/bin/bkms-server

# 配置文件目录
RUN mkdir -p /app/etc

COPY --from=builder /go/src/configs/config.yaml /app/etc/config.yaml

RUN mkdir -p /app/assets

# 流水线模板
COPY --from=builder /go/src/pkg/bkintegrations/bkci/assets/pipeline_templates /app/assets/pipeline_templates

# 组件配置
COPY --from=builder /go/src/pkg/extension/component/assets/comps /app/assets/comps

# 集群插件配置
COPY --from=builder /go/src/pkg/core/env/clusteraddon/assets/addons /app/assets/addons

# 日志目录
RUN mkdir -p /app/logs

# command + args example:
# - bkms-server webserver --srvCfg /app/etc/config.yaml
# - bkms-server worker --srvCfg /app/etc/config.yaml
