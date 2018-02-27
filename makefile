# Helper makefile for evilproxy

# Version derived from git tags with describe and injected
# into main.version via linker magic
VER=$(shell git describe --dirty)
ARGS= -ldflags "-X main.version=$(VER)"

# build builds evpx
build:
	go build $(ARGS) ./cmd/evpx

# build-all builds targets for all os/arch tuples
build-all:
	gox -output=build/{{.OS}}-{{.Arch}}/{{.Dir}} $(ARGS) ./cmd/...

# deps fetches dependencies for evilproxy dev
deps:
	go get -u github.com/golang/dep/cmd/dep
	go get github.com/mitchellh/gox
	dep ensure

# install installs evpx into $GOPATH/bin
install:
	go install ./cmd/...

# clean sorts out the multiarch build dir
clean:
	rm -rf build/*


.PHONY: build build-all package install
