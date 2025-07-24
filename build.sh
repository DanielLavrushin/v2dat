#!/usr/bin/env bash
set -e

PKG=github.com/urlesistiana/v2dat
VERSION=$(git -C "$(go env GOPATH)/pkg/mod/$PKG" 2>/dev/null || echo dev)

build() {
    GOOS=linux GOARCH=$1 GOARM=$2 GOMIPS=$3 \
        go build -tags "protobuf,unsafe" -trimpath -ldflags "-s -w -X main.version=$VERSION" \
        -o dist/$4/v2dat ./
}

build_windows() {
    GOOS=windows GOARCH=$1 GOARM=$2 GOMIPS=$3 \
        go build -tags "protobuf,unsafe" -trimpath -ldflags "-s -w -X main.version=$VERSION" \
        -o dist/$4/v2dat.exe ./
}

build_macos() {
    GOOS=darwin GOARCH=$1 GOARM=$2 GOMIPS=$3 \
        go build -tags "protobuf,unsafe" -trimpath -ldflags "-s -w -X main.version=$VERSION" \
        -o dist/$4/v2dat ./
}

archive(){
    NAME=$1 V2DATDIR=$2 
    mkdir -p dist/assets
     tar -czf dist/assets/${NAME}.tar.gz -C ${V2DATDIR} .
}

rm -rf dist && mkdir dist

build arm 5 "" armv5
archive v2dat-armv5 ./dist/armv5
archive v2dat ./dist/armv5

build arm 7 "" armv7
archive v2dat-armv7 ./dist/armv7

build arm64 "" "" arm64
archive v2dat-arm64 ./dist/arm64

build arm64 8 "" arm64v8
archive v2dat-arm64v8 ./dist/arm64v8

# build for linux on x86_64
build amd64 "" "" amd64
archive v2dat-amd64 ./dist/amd64

# build for linux on 386
build 386 "" "" i386
archive v2dat-i386 ./dist/i386

# build for linux on riscv64
build riscv64 "" "" riscv64
archive v2dat-riscv64 ./dist/riscv64

# mips soft-float – very old routers that Merlin still supports
build mipsle "" softfloat mipsle
archive v2dat-mipsle ./dist/mipsle

# mips hard-float
build mips64le "" "" mips64le
archive v2dat-mips64le ./dist/mips64le

# mips64 hard-float
build mips64 "" "" mips64
archive v2dat-mips64 ./dist/mips64

# windows on x86_64
build_windows amd64 "" "" win-amd64
archive v2dat-win-amd64 ./dist/win-amd64

# windows on 386
build_windows 386 "" "" win-i386
archive v2dat-win-i386 ./dist/win-i386

# windows on arm64
build_windows arm64 "" "" win-arm64
archive v2dat-win-arm64 ./dist/win-arm64

# macOS on x86_64
build_macos amd64 "" "" macos-amd64
archive v2dat-macos-amd64 ./dist/macos-amd64

# macOS on arm64
build_macos arm64 "" "" macos-arm64
archive v2dat-macos-arm64 ./dist/macos-arm64