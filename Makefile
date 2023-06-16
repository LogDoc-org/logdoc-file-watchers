PROJECTNAME=$(shell basename "$(PWD)")

MAC_ARCH=arm64
ARCH=amd64

VERSION=$(shell git describe --tags --always --long --dirty)
WINDOWS=$(PROJECTNAME)_windows_$(ARCH)_$(VERSION).exe
LINUX=$(PROJECTNAME)_linux_$(ARCH)_$(VERSION)
DARWIN=$(PROJECTNAME)_darwin_$(ARCH)_$(VERSION)

# Go переменные.
GOBASE="/usr/local/go"
GOPATH=$(shell /usr/local/go/bin/go env GOPATH)
GOBIN=$(GOBASE)/bin
GOFILES=$(wildcard *.go)

# Перенаправление вывода ошибок в файл, чтобы мы показывать его в режиме разработки.
STDERR=./tmp/.$(PROJECTNAME)-stderr.txt

# PID-файл будет хранить идентификатор процесса, когда он работает в режиме разработки
PID=./tmp/.$(PROJECTNAME)-api-server.pid

# Make пишет работу в консоль Linux. Сделаем его silent.
MAKEFLAGS += --silent

.PHONY: all test clean

all: test build

test:
	@echo "Running tests..."
	@go test ./...

build: windows linux darwin
	@echo version: $(VERSION)

windows: $(WINDOWS)

linux: $(LINUX)

darwin: $(DARWIN)

$(WINDOWS):
	@echo "Building windows app..."
	@env GOOS=windows GOARCH=$(ARCH) go build -v -o bin/$(WINDOWS) -ldflags="-s -w -X main.version=$(VERSION)" ./cmd/main.go

$(LINUX):
	@echo "Building linux app..."
	@env GOOS=linux GOARCH=$(ARCH) go build -v -o bin/$(LINUX) -ldflags="-s -w -X main.version=$(VERSION)" ./cmd/main.go

$(DARWIN):
	@echo "Building macos app..."
	@env GOOS=darwin GOARCH=$(MAC_ARCH) go build -v -o bin/$(DARWIN) -ldflags="-s -w -X main.version=$(VERSION)" ./cmd/main.go

clean:
	@echo "Cleaning up..."
	@go clean
	@rm -f ./bin/$(WINDOWS) ./bin/$(LINUX) ./bin/$(DARWIN)