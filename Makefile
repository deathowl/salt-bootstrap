BINARY=salt-bootstrap

VERSION=0.1.0
BUILD_TIME=$(shell date +%FT%T)
LDFLAGS=-ldflags "-X github.com/sequenceiq/salt-bootstrap/saltboot.Version=${VERSION} -X github.com/sequenceiq/salt-bootstrap/saltboot.BuildTime=${BUILD_TIME}"

deps:
	go get github.com/gliderlabs/glu

build:
	GOOS=linux go build -a -installsuffix cgo ${LDFLAGS} -o build/Linux/${BINARY} main.go
	GOOS=darwin go build -a -installsuffix cgo ${LDFLAGS} -o build/Darwin/${BINARY} main.go
	
release: build
	rm -rf release
	glu release

.PHONY: build
