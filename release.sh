#!/bin/bash -ex

mkdir -p bin

# Linux amd64
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o bin/ab-linux-amd64

# Linux arm64
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "-s -w" -o bin/ab-linux-arm64

# Windows (x86_64)
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o bin/ab.exe

# macOS (Apple Silicon)
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags "-s -w" -o bin/ab-darwin-arm64

# FreeBSD (x86_64)
CGO_ENABLED=0 GOOS=freebsd GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o bin/ab-freebsd-amd64

cd bin
sha256sum ab* > CHECKSUMS
