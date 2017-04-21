# Makefile to help building go components

COMMAND_NAME=photon
VERSION=$(shell cat VERSION)
LDFLAGS="-X main.githash=`git rev-parse --short HEAD` -X main.commandName=$(COMMAND_NAME) -X main.version=$(VERSION)"
GOBUILD=go build -ldflags $(LDFLAGS)

all: test build binaries

binaries: darwin/amd64 windows/amd64 linux/amd64

darwin/amd64:
	$(eval export GOOS=darwin)
	$(eval export GOARCH=amd64)
	$(eval export fileext=)
	make build

windows/amd64:
	$(eval export GOOS=windows)
	$(eval export GOARCH=amd64)
	$(eval export fileext=.exe)
	make build

linux/amd64:
	$(eval export GOOS=linux)
	$(eval export GOARCH=amd64)
	$(eval export fileext=)
	make build

# go build arch is controlled by env var GOOS and GOARCH, when not set it use current machine native arch
build:
	$(GOBUILD) -o bin/$(GOOS)$(GOARCH)/$(COMMAND_NAME)$(fileext) ./photon

#
# get the tools
#
tools:
	go get -u github.com/toliaqat/errcheck
	go get -u golang.org/x/tools/cmd/goimports
	go get -u github.com/golang/lint/golint
	go get -u github.com/tools/godep
	go get -u github.com/kardianos/govendor

test: tools
	errcheck ./photon/...
	go vet `go list ./... | grep -v '/vendor/'`
	golint
	chmod +x ci/check-fmt.sh
	./ci/check-fmt.sh
	go test -v ./...
