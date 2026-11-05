.PHONY: all run test helper build bundle dmg dmg-all clean benchmark check-update upgrade

all: build

check-update:
	@echo "==> 检查 mygo 最新版本..."
	go list -m -u github.com/egoist/mygo

upgrade:
	@echo "==> 升级 mygo 库与 CLI 工具至最新版..."
	go get -u github.com/egoist/mygo@latest
	go get -tool github.com/egoist/mygo/cmd/mygo@latest
	go mod tidy
	@echo "==> 执行自测校验..."
	$(MAKE) test

run:
	CGO_ENABLED=0 go run ./cmd/clover

# 预编译原生能力 helper（PDF/水印/Finder sfl3）；运行时无需 Xcode CLT。
helper:
	mkdir -p build
	swiftc -O -o build/clover-helper tools/clover-helper/main.swift

test: helper
	CLOVER_HELPER_BIN=$(CURDIR)/build/clover-helper CGO_ENABLED=0 go test -race ./...

benchmark:
	CGO_ENABLED=0 go test -benchmem -run=^$$ -bench=. ./...

build:
	mkdir -p build
	CGO_ENABLED=0 go build -ldflags="-s -w" -o build/clover ./cmd/clover

bundle:
	chmod +x scripts/package.sh
	./scripts/package.sh app

dmg:
	chmod +x scripts/package.sh
	./scripts/package.sh dmg

dmg-all:
	chmod +x scripts/package.sh
	ARCH=arm64 ./scripts/package.sh dmg
	ARCH=x86_64 ./scripts/package.sh dmg

clean:
	rm -rf build
