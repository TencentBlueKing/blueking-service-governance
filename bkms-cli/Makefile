.DEFAULT_GOAL:=help

VERSION ?= $(patsubst v%,%,$(notdir $(shell git describe --tags --abbrev=0 --match 'bkms-cli/v*' 2>/dev/null || echo 0.0.0-dev)))
GIT_HASH ?= $(shell git rev-parse HEAD)
GOOS ?= ""
GOARCH ?= ""
REPO_ROOT := $(shell git rev-parse --show-toplevel)

ifeq (${GOOS}, windows)
    BIN_NAME=bkms-cli.exe
else
    BIN_NAME=bkms-cli
endif

## 依赖工具下载位置
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

BUILD_TIME=${shell date +%Y-%m-%dT%H:%M:%S%z}

# 更新源：GitHub Releases 的 owner/repository
UPDATE_SOURCE ?= TencentBlueKing/blueking-service-governance

LDFLAGS="-s -w \
	-X github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/version.Version=${VERSION} \
	-X github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/version.GitHash=${GIT_HASH} \
	-X github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/version.BuildTime=${BUILD_TIME} \
	-X github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/updater.updateSource=${UPDATE_SOURCE}"

GORELEASER_ENV = \
	VERSION=$(VERSION) \
	UPDATE_SOURCE=$(UPDATE_SOURCE)

.PHONY: help
help: ## 展示可用 make 命令及说明
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ 开发/构建命令

.PHONY: build
build: tidy ## 构建 bkms-cli 可执行文件（当前平台或指定 GOOS/GOARCH）
	@mkdir -p ./build/
	@if [ -z "${GOOS}" ] || [ "${GOOS}" = '""' ]; then \
		CGO_ENABLED=0 go build -ldflags ${LDFLAGS} -o ./build/${BIN_NAME} ./main.go; \
	else \
		$(eval EXT := $(if $(filter windows,${GOOS}),.exe,)) \
		CGO_ENABLED=0 GOOS=${GOOS} GOARCH=${GOARCH} go build -ldflags ${LDFLAGS} \
			-o ./build/bkms-cli-${GOOS}-${GOARCH}${EXT} ./main.go; \
	fi

.PHONY: build-all
build-all: tidy goreleaser ## 用 goreleaser 构建所有平台（产物在 dist/）
	cd $(REPO_ROOT) && $(GORELEASER_ENV) $(GORELEASER) build \
		--config bkms-cli/.goreleaser.yaml \
		--snapshot --clean --skip=validate
	@ls -lh ./dist/ || true

.PHONY: clean
clean: ## 清理所有构建产物
	rm -rf ./build/* ./dist

.PHONY: vet
vet: ## 执行 go vet ./...
	go vet ./...

.PHONY: tidy
tidy: ## 执行 go mod tidy
	go mod tidy

.PHONY: test
test: ginkgo ## 执行 ginkgo 单元测试
	$(GINKGO) --cover --coverprofile cover.out ./pkg/...

.PHONY: fmt
fmt: install-golangci-lint  ## 执行代码格式化
	$(GOCILINT) fmt .

.PHONY: lint
lint: install-golangci-lint  ## 执行代码检查
	$(GOCILINT) run

##@ E2E 功能测试命令

.PHONY: e2e-build
e2e-build: tidy ## 构建 E2E 测试专用二进制（bkms-cli-e2e），避免覆盖正式构建产物
	@mkdir -p ./build/
	CGO_ENABLED=0 go build -ldflags ${LDFLAGS} -o ./build/bkms-cli-e2e ./main.go

.PHONY: e2e-go-test
e2e-go-test: e2e-build ginkgo ## 构建 E2E 专用二进制并执行 Go E2E 测试（自动加载 test/e2e/.env）
	@if [ -f ./test/e2e/.env ]; then \
		echo "加载环境变量: test/e2e/.env"; \
		set -a && . ./test/e2e/.env && set +a && \
		BKMS_CLI_BIN=$(shell pwd)/build/bkms-cli-e2e \
		BKMS_CLI_BUILD_DIR=$(shell pwd)/build \
		$(GINKGO) -v --fail-fast ./test/e2e/...; \
	else \
		echo "未找到 test/e2e/.env，使用当前 shell 环境变量"; \
		BKMS_CLI_BIN=$(shell pwd)/build/bkms-cli-e2e \
		BKMS_CLI_BUILD_DIR=$(shell pwd)/build \
		$(GINKGO) -v --fail-fast ./test/e2e/...; \
	fi

##@ 开发工具安装命令

GINKGO ?= $(LOCALBIN)/ginkgo
GINKGO_VERSION ?= v2.32.0

GOCILINT ?= $(LOCALBIN)/golangci-lint
GOCILINT_VERSION ?= v2.6.2

GORELEASER ?= $(LOCALBIN)/goreleaser
GORELEASER_VERSION ?= v2.17.1

.PHONY: ginkgo
ginkgo: | $(LOCALBIN)  ## Install ginkgo tool
	@if ! (test -s $(GINKGO) && $(GINKGO) version | grep -q "Ginkgo Version $(patsubst v%,%,$(GINKGO_VERSION))"); then \
		GOBIN=$(LOCALBIN) go install github.com/onsi/ginkgo/v2/ginkgo@$(GINKGO_VERSION); \
	fi

.PHONY: install-golangci-lint
install-golangci-lint: $(GOCILINT) ## 下载 golangci-lint 二进制
$(GOCILINT): $(LOCALBIN)
	@if ! (test -s $(GOCILINT) && $(GOCILINT) --version | grep -q "$(patsubst v%,%,$(GOCILINT_VERSION))"); then \
		rm -f $(GOCILINT); \
		GOBIN=$(LOCALBIN) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOCILINT_VERSION); \
	fi

.PHONY: goreleaser
goreleaser: | $(LOCALBIN) ## 下载 goreleaser 二进制
	@if ! (test -s $(GORELEASER) && $(GORELEASER) --version 2>/dev/null | grep -q "$(patsubst v%,%,$(GORELEASER_VERSION))"); then \
		rm -f $(GORELEASER); \
		GOBIN=$(LOCALBIN) go install github.com/goreleaser/goreleaser/v2@$(GORELEASER_VERSION); \
	fi
