.DEFAULT_GOAL:=help

VERSION ?= $(shell git describe --tags --abbrev=0)
GIT_HASH ?= $(shell git rev-parse HEAD)
GOOS ?= ""
GOARCH ?= ""

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

# UPX 压缩配置（用于 make compress）
# UPX_LEVEL: 压缩级别 1-9，数字越大压缩率越高但速度越慢，默认为 9（最佳压缩）
UPX_LEVEL ?= 9
UPX ?= $(shell command -v upx 2>/dev/null)

# 优化编译参数以减小二进制文件大小
# -s: 去除符号表
# -w: 去除 DWARF 调试信息

# BCS API 网关地址，构建时必须通过 BCS_API_HOST=xxx 传入
BCS_API_HOST ?=

# bkms_server 服务地址，作为配置文件中 bkmsBaseUrl 的值，构建时必须通过 BKMS_BASE_URL=xxx 传入
BKMS_BASE_URL ?=

.PHONY: check-build-vars
check-build-vars: ## 校验构建所需变量
	@if [ -z "$(BCS_API_HOST)" ]; then \
		echo "错误: 必须传入 BCS_API_HOST，例如: make build BCS_API_HOST=<bcs-api-host> BKMS_BASE_URL=<bkms-base-url>"; \
		exit 1; \
	fi
	@if [ -z "$(BKMS_BASE_URL)" ]; then \
		echo "错误: 必须传入 BKMS_BASE_URL，例如: make build BCS_API_HOST=<bcs-api-host> BKMS_BASE_URL=<bkms-base-url>"; \
		exit 1; \
	fi

LDFLAGS="-s -w \
	-X github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/version.Version=${VERSION} \
	-X github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/version.GitHash=${GIT_HASH} \
	-X github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/version.BuildTime=${BUILD_TIME} \
	-X github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/publish.bcsAPIHost=${BCS_API_HOST} \
	-X github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/config.bkmsBaseURL=${BKMS_BASE_URL}"

## 支持的平台列表
PLATFORMS := \
	linux/amd64 \
	linux/arm64 \
	darwin/amd64 \
	darwin/arm64 \
	windows/amd64 \
	windows/arm64

## 各操作系统支持的架构列表
LINUX_PLATFORMS := linux/amd64 linux/arm64
DARWIN_PLATFORMS := darwin/amd64 darwin/arm64
WINDOWS_PLATFORMS := windows/amd64 windows/arm64

## 通用平台构建函数
## 参数: $(1) = 平台列表, $(2) = 平台显示名称
define build-platforms
	@echo "构建 $(2) 平台..."
	@mkdir -p ./build/
	@$(foreach platform,$(1), \
		$(eval OS := $(word 1,$(subst /, ,$(platform)))) \
		$(eval ARCH := $(word 2,$(subst /, ,$(platform)))) \
		$(eval EXT := $(if $(filter windows,$(OS)),.exe,)) \
		echo "构建 $(OS)/$(ARCH)..." && \
		CGO_ENABLED=0 GOOS=$(OS) GOARCH=$(ARCH) go build -ldflags ${LDFLAGS} \
			-o ./build/bkms-cli-$(OS)-$(ARCH)$(EXT) ./main.go && \
		echo "✓ $(OS)/$(ARCH) 构建完成: bkms-cli-$(OS)-$(ARCH)$(EXT)" || exit 1; \
	)
	@echo "✓ $(2) 平台构建完成"
	@ls -lh ./build/bkms-cli-$(word 1,$(subst /, ,$(firstword $(1))))-*
endef

.PHONY: help
help: ## 展示可用 make 命令及说明
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ 开发/构建命令

.PHONY: build
build: check-build-vars tidy ## 构建 bkms-cli 可执行文件（当前平台或指定 GOOS/GOARCH）
	@mkdir -p ./build/
	@if [ -z "${GOOS}" ] || [ "${GOOS}" = '""' ]; then \
		CGO_ENABLED=0 go build -ldflags ${LDFLAGS} -o ./build/${BIN_NAME} ./main.go; \
	else \
		$(eval EXT := $(if $(filter windows,${GOOS}),.exe,)) \
		CGO_ENABLED=0 GOOS=${GOOS} GOARCH=${GOARCH} go build -ldflags ${LDFLAGS} \
			-o ./build/bkms-cli-${GOOS}-${GOARCH}${EXT} ./main.go; \
	fi

.PHONY: build-all
build-all: check-build-vars tidy ## 构建所有支持平台的二进制文件（命名格式: bkms-cli-{os}-{arch}）
	@echo "开始构建所有平台的二进制文件..."
	@mkdir -p ./build/
	@$(foreach platform,$(PLATFORMS), \
		$(eval OS := $(word 1,$(subst /, ,$(platform)))) \
		$(eval ARCH := $(word 2,$(subst /, ,$(platform)))) \
		$(eval EXT := $(if $(filter windows,$(OS)),.exe,)) \
		echo "构建 $(OS)/$(ARCH)..." && \
		CGO_ENABLED=0 GOOS=$(OS) GOARCH=$(ARCH) go build -ldflags ${LDFLAGS} \
			-o ./build/bkms-cli-$(OS)-$(ARCH)$(EXT) ./main.go && \
		echo "✓ $(OS)/$(ARCH) 构建完成: bkms-cli-$(OS)-$(ARCH)$(EXT)" || exit 1; \
	)
	@echo ""
	@echo "所有平台构建完成！"
	@echo "构建产物位置: ./build/"
	@ls -lh ./build/

.PHONY: build-linux
build-linux: check-build-vars tidy ## 构建 Linux 平台的二进制文件（amd64 + arm64）
	$(call build-platforms,$(LINUX_PLATFORMS),Linux)

.PHONY: build-darwin
build-darwin: check-build-vars tidy ## 构建 macOS 平台的二进制文件（amd64 + arm64）
	$(call build-platforms,$(DARWIN_PLATFORMS),macOS)

.PHONY: build-windows
build-windows: check-build-vars tidy ## 构建 Windows 平台的二进制文件（amd64 + arm64）
	$(call build-platforms,$(WINDOWS_PLATFORMS),Windows)

.PHONY: clean
clean: ## 清理所有构建产物
	@echo "清理构建产物..."
	rm -rf ./build/*
	@echo "✓ 清理完成"

.PHONY: compress
compress: ## 使用 UPX 压缩 build 目录下的二进制文件（仅支持 linux 和 windows/amd64）
	@if [ -z "$(UPX)" ]; then \
		echo "错误: 未找到 UPX，请先安装 UPX"; \
		echo "  Ubuntu/Debian: sudo apt-get install upx"; \
		echo "  macOS: brew install upx"; \
		echo "  或访问: https://github.com/upx/upx/releases"; \
		exit 1; \
	fi
	@echo "使用 UPX 压缩 build 目录下的二进制文件..."
	@echo "注意: 仅支持 Linux (amd64/arm64) 和 Windows (amd64)"
	@echo ""
	@for file in ./build/bkms-cli*; do \
		if [ -f "$$file" ]; then \
			case "$$file" in \
				*linux*) \
					echo "压缩 $$file..." && \
					$(UPX) -$(UPX_LEVEL) -q "$$file" && \
					echo "✓ $$file 压缩完成";; \
				*windows-amd64*) \
					echo "压缩 $$file..." && \
					$(UPX) -$(UPX_LEVEL) -q "$$file" && \
					echo "✓ $$file 压缩完成";; \
				*darwin*) \
					echo "⚠ 跳过 $$file (UPX 不支持 macOS)";; \
				*windows-arm64*) \
					echo "⚠ 跳过 $$file (UPX 不支持 windows/arm64)";; \
				*) \
					echo "⚠ 跳过 $$file (未知平台)";; \
			esac; \
		fi; \
	done
	@echo ""
	@echo "压缩完成！"
	@ls -lh ./build/

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
e2e-build: check-build-vars tidy ## 构建 E2E 测试专用二进制（bkms-cli-e2e），避免覆盖正式构建产物
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
