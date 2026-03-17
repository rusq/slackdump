SHELL=/bin/sh

CMD=./cmd/slackdump
OUTPUT=slackdump
EXECUTABLE=slackdump
BUILD=$(shell git describe --tags)
COMMIT=$(shell git rev-parse --short HEAD)
ifeq ($(BUILD_DATE),)
	BUILD_DATE=$(shell TZ=UTC date -u '+%Y-%m-%d %H:%M:%SZ')
endif

PKG=github.com/rusq/slackdump/v4

# Use podman if docker is not available
CONTAINER_CMD=$(shell command -v docker podman 2>/dev/null | head -1)

LDFLAGS="-s -w -X 'main.commit=$(COMMIT)' -X 'main.version=$(BUILD)' -X 'main.date=$(BUILD_DATE)'"
LDFLAGS_DEBUG="-X 'main.commit=$(COMMIT)' -X 'main.version=$(BUILD)' -X 'main.date=$(BUILD_DATE)'"
OSES=linux darwin windows
DISTFILES=README.md LICENSE
ZIPFILES=$(foreach s,$(OSES),$(OUTPUT)-$s.zip)


.PHONY: dist all

# special guest.
$(OUTPUT)-windows.zip: EXECUTABLE=$(OUTPUT).exe

$(foreach s,$(OSES),$(eval $(OUTPUT)-$s.zip: GOOS=$s))
$(foreach s,$(OSES),$(eval $(OUTPUT)-$s.zip: $(EXECUTABLE)))

# rules
%.ps: %.1
	man -t ./$< > $@

%.pdf: %.ps
	ps2pdf $< $@

all: ## Build the executable (incremental - Go handles changes)
	GOARCH=$(GOARCH) GOOS=$(GOOS) go build -ldflags=$(LDFLAGS) -o $(EXECUTABLE) $(CMD)

dist: ## Build distribution archives for all platforms
	$(MAKE) $(ZIPFILES)

%.zip:
	7z a $@ $(EXECUTABLE) $(DISTFILES)
	rm $(EXECUTABLE)


$(OUTPUT).exe: GOOS=windows
$(OUTPUT).exe: $(OUTPUT)

$(OUTPUT):
	GOARCH=$(GOARCH) GOOS=$(GOOS) go build -ldflags=$(LDFLAGS) -o $(EXECUTABLE) $(CMD)

debug: ## Build with debug symbols (no stripping)
	GOARCH=$(GOARCH) GOOS=$(GOOS) go build -ldflags=$(LDFLAGS_DEBUG) -o $(EXECUTABLE) $(CMD)
.PHONY: debug

x86_%:
	GOARCH=amd64 go build -ldflags=$(LDFLAGS) -o $@ $(CMD)

arm_%:
	GOARCH=arm64 go build -ldflags=$(LDFLAGS) -o $@ $(CMD)


clean: ## Remove built artifacts
	-rm slackdump slackdump.exe $(wildcard *.zip)
	-rm -rf slackdump_$(shell date +%Y)*
.PHONY: clean

# Via http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
help: ## help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
.PHONY: help

docker_test: ## Build container image for testing
	$(CONTAINER_CMD) build .

callvis: ## Generate call graph visualization
	go-callvis -group pkg,type -limit $(PKG) $(PKG)/cmd/slackdump

goreleaser: ## Run goreleaser for snapshot release
	goreleaser check
	goreleaser release --snapshot --clean

tags: ## Generate tags file for editors
	gotags -R *.go > $@

generate: | install_tools ## Run go generate for all packages
	go generate ./...
.PHONY:generate

install_tools: ## Install required development tools (mockgen, stringer)
	go install go.uber.org/mock/mockgen@latest
	go install golang.org/x/tools/cmd/stringer@latest
.PHONY: install_tools

slackdump.pdf: slackdump.ps

slackdump.ps: slackdump.1

# =============================================================================
# Test targets
# =============================================================================

test: ## Run tests
	go test -race -cover ./...
.PHONY: test

vet: ## Run go vet
	go vet ./...
.PHONY: vet

fmt: ## Check code formatting
	@gofmt -d .
.PHONY: fmt

lint: ## Run golangci-lint
	golangci-lint run ./...
.PHONY: lint

aurtest: ## Test AUR package build flags
	GOFLAGS="-buildmode=pie -trimpath -ldflags=-linkmode=external -mod=readonly -modcacherw" go build -o 'deleteme' ./cmd/slackdump
	rm deleteme
.PHONY: aurtest

test-all: fmt vet test aurtest lint ## Run all tests (fmt, vet, tests, AUR build, lint (slow))
.PHONY: test-all