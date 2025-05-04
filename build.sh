#!/usr/bin/env bash
set -e

PKG=github.com/urlesistiana/v2dat
VERSION=$(git -C "$(go env GOPATH)/pkg/mod/$PKG" 2>/dev/null || echo dev)

build() {
    GOOS=linux GOARCH=$1 GOARM=$2 GOMIPS=$3 \
        go build -trimpath -ldflags "-s -w -X main.version=$VERSION" \
        -o dist/$4/v2dat ./
}

rm -rf dist && mkdir dist

build arm 5 "" armv5

build arm 7 "" armv7

build arm64 "" "" arm64

# mips soft-float – very old routers that Merlin still supports
build mipsle "" softfloat mipsle
