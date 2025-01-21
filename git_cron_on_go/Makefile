GOOS ?= linux
GOARCH ?= amd64
BINARY_NAME = github-sync
CGO_ENABLED ?= 0
MODULE_NAME = repo-sync # Имя модуля

all: mod-init mod-tidy cross-compile

mod-init:
	@if [ ! -f go.mod ]; then \
	go mod init $(MODULE_NAME); \
	fi

mod-tidy:
	go mod tidy

build:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags="-s -w" -o $(BINARY_NAME)-$(GOOS)-$(GOARCH) main.go

clean:
	rm -f $(BINARY_NAME)-*

# Cross-compilation targets
linux-amd64:
	@$(MAKE) build GOOS=linux GOARCH=amd64
# windows-amd64:
#       @$(MAKE) build GOOS=windows GOARCH=amd64
# darwin-amd64:
#       @$(MAKE) build GOOS=darwin GOARCH=amd64
linux-arm64:
	@$(MAKE) build GOOS=linux GOARCH=arm64


cross-compile: linux-amd64 linux-arm64 #windows-amd64 darwin-amd64
