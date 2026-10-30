VERSION ?= $(shell git describe --always)
GIT_HASH ?= $(shell git rev-parse HEAD)

GOPROXY ?= https://goproxy.cn,direct
GOSUMDB ?= off

WORKSPACE=$(shell pwd)
BUILD_TIME=${shell date -u +%Y-%m-%dT%H:%M:%SZ}
GO_BUILD=CGO_ENABLED=0 go build -trimpath

# build path config
export PACKAGE_PATH=${WORKSPACE}/build
export CONFIGS_PATH=${WORKSPACE}/configs

LDFLAGS="-X github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/version.Version=${VERSION} \
	-X github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/version.GitHash=${GIT_HASH} \
	-X github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/version.BuildTime=${BUILD_TIME}"

.DEFAULT_GOAL := help

.PHONY: help
help:  ## Display available make commands
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

##@ Development

.PHONY: fmt
fmt: golangci-lint  ## Format the code
	$(GOCILINT) fmt .

.PHONY: lint
lint: golangci-lint  ## Lint the code
	$(GOCILINT) run

.PHONY: test
test: ginkgo  ## Run the tests
	# Check BKMS_SERVER_CONFIG_PATH since it's required for unit tests.
ifndef BKMS_SERVER_CONFIG_PATH
	$(error "BKMS_SERVER_CONFIG_PATH is not set. Please set it to the path of the test configuration file.")
endif
	$(GINKGO) -gcflags="all=-l -N" --cover --coverprofile cover.out ./...

.PHONY: test-multi-tenant
test-multi-tenant:  ## Run the tests in multi-tenant mode (platform tenant is system)
	BKMS_TEST_MULTI_TENANT=true $(MAKE) test

.PHONY: apidocs
apidocs: swag  ## Generate Swagger docs for Gin APIs
	$(SWAG) init -g main.go -o ./docs/apis --parseInternal --outputTypes go,json,yaml

##@ Build

.PHONY: build
build:  ## Build the webserver binary
	mkdir -p ${PACKAGE_PATH}
	${GO_BUILD} -ldflags ${LDFLAGS} -o ${PACKAGE_PATH}/bkms-server main.go

.PHONY: build-dev
build-dev:  ## Build the webserver binary for development
	mkdir -p ${PACKAGE_PATH}
	CGO_ENABLED=0 go build -gcflags="all=-N -l" -ldflags ${LDFLAGS} -o ${PACKAGE_PATH}/bkms-server main.go

.PHONY: docker
docker:  ## Build the docker image (context is repo root)
	docker build -f ./Dockerfile \
	--build-arg VERSION=${VERSION} \
	--build-arg GIT_HASH=${GIT_HASH} \
	--build-arg GOPROXY=${GOPROXY} \
	--build-arg GOSUMDB=${GOSUMDB} \
	-t bkms-server:${VERSION} \
	../..

##@ Dependencies

# Tool binaries
# All tools used by current project are installed in $(LOCALBIN)
GINKGO ?= $(LOCALBIN)/ginkgo
GINKGO_VERSION ?= v2.32.0

# formatter and linter
GOCILINT ?= $(LOCALBIN)/golangci-lint
GOCILINT_VERSION ?= v2.4.0

# swagger cmdline tools
SWAG ?= $(LOCALBIN)/swag
SWAG_VERSION ?= v1.16.6

# database migration tool
MIGRATE ?= $(LOCALBIN)/migrate
MIGRATE_VERSION ?= v4.18.3

.PHONY: ginkgo
ginkgo: | $(LOCALBIN)  ## Install ginkgo tool
	@if ! (test -s $(GINKGO) && $(GINKGO) version | grep -q "Ginkgo Version $(patsubst v%,%,$(GINKGO_VERSION))"); then \
		GOBIN=$(LOCALBIN) go install github.com/onsi/ginkgo/v2/ginkgo@$(GINKGO_VERSION); \
	fi

.PHONY: golangci-lint
golangci-lint: | $(LOCALBIN)  ## Install golangci-lint tool
	@if ! (test -s $(GOCILINT) && $(GOCILINT) --version | grep -q "version $(patsubst v%,%,$(GOCILINT_VERSION))"); then \
		GOBIN=$(LOCALBIN) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOCILINT_VERSION); \
	fi

.PHONY: swag
swag: | $(LOCALBIN) ## Install swag tool
	# @if ! (test -s $(SWAG) && $(SWAG) --version 2>/dev/null | grep -q "swag version $(SWAG_VERSION)"); then \
	# Workaround: swag v1.16.6 reports v1.16.4 from --version.
	@if ! (test -s $(SWAG) && go version -m $(SWAG) 2>/dev/null | awk '$$1 == "mod" && $$2 == "github.com/swaggo/swag" && $$3 == "$(SWAG_VERSION)" { found = 1 } END { exit !found }'); then \
		GOBIN=$(LOCALBIN) go install github.com/swaggo/swag/cmd/swag@$(SWAG_VERSION); \
	fi

.PHONY: migrate_binary
migrate_binary: | $(LOCALBIN)  ## Install golang-migrate tool
	@if ! (test -s $(MIGRATE) && go version -m $(MIGRATE) 2>/dev/null | awk '$$1 == "mod" && $$2 == "github.com/golang-migrate/migrate/v4" && $$3 == "$(MIGRATE_VERSION)" { found = 1 } END { exit !found }'); then \
		GOBIN=$(LOCALBIN) go install -tags 'mongodb' github.com/golang-migrate/migrate/v4/cmd/migrate@$(MIGRATE_VERSION); \
	fi
