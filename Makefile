export GO111MODULE := on
export CGO_ENABLED=1

OS_NAME = $(shell uname -s | tr A-Z a-z)

# OpenNHP submodule directory
OPENNHP_DIR = third_party/opennhp

all: generate-version-and-build


generate-version-and-build:
	@echo "[StealthDNS] Start building..."
	@$(MAKE) init
	@$(MAKE) build-sdk
	@$(MAKE) build
	@echo "[StealthDNS] Build for platform ${OS_NAME} successfully done!"

init:
	@echo "[StealthDNS] Initializing..."
	git clean -df release
	git submodule update --init --recursive
	go mod tidy

# Build OpenNHP SDK from submodule
build-sdk:
	@echo "[StealthDNS] Building OpenNHP SDK from submodule..."
ifeq ($(OS_NAME), linux)
	@$(MAKE) build-sdk-linux
else ifeq ($(OS_NAME), darwin)
	@$(MAKE) build-sdk-macos
else
	@echo "[StealthDNS] Skipping SDK build on ${OS_NAME}, use build.bat for Windows"
endif

build-sdk-linux:
	@echo "[StealthDNS] Building Linux SDK (nhp-agent.so)..."
	cd $(OPENNHP_DIR)/nhp && go mod tidy
	cd $(OPENNHP_DIR)/endpoints && go mod tidy
	cd $(OPENNHP_DIR)/endpoints && \
		go build -a -trimpath -buildmode=c-shared -ldflags="-w -s" -v \
		-o ../../sdk/nhp-agent.so ./agent/main/main.go ./agent/main/export.go
	@echo "[StealthDNS] Linux SDK built successfully!"

build-sdk-macos:
	@echo "[StealthDNS] Building macOS SDK (nhp-agent.dylib)..."
	cd $(OPENNHP_DIR)/nhp && go mod tidy
	cd $(OPENNHP_DIR)/endpoints && go mod tidy
	cd $(OPENNHP_DIR)/endpoints && \
		GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 \
		go build -a -trimpath -buildmode=c-shared -ldflags="-w -s" -v \
		-o ../../sdk/nhp-agent.dylib ./agent/main/main.go ./agent/main/export.go
	@echo "[StealthDNS] macOS SDK built successfully!"

build:
	@echo "[StealthDNS] Building package..."
	go build -trimpath -ldflags="-w -s" -v -o ./release/stealth-dns ./main.go && \
	cp ./etc/*.toml ./release/etc/ && \
	cp ./sdk/nhp-agent.* ./release/sdk/ && \
	cp ./etc/cert/rootCA.pem ./release/etc/cert/
ifeq ($(OS_NAME), darwin)
	install_name_tool -change nhp-agent.dylib ./sdk/nhp-agent.dylib ./release/stealth-dns
endif

# Windows SDK build (call from PowerShell/CMD)
build-sdk-windows:
	@echo "[StealthDNS] Building Windows SDK (nhp-agent.dll)..."
	cd $(OPENNHP_DIR)/nhp && go mod tidy
	cd $(OPENNHP_DIR)/endpoints && go mod tidy
	cd $(OPENNHP_DIR)/endpoints && \
		go build -a -trimpath -buildmode=c-shared -ldflags="-w -s" -v \
		-o ../../sdk/nhp-agent.dll ./agent/main/main.go ./agent/main/export.go
	@echo "[StealthDNS] Windows SDK built successfully!"

# Clean SDK binaries
clean-sdk:
	@echo "[StealthDNS] Cleaning SDK binaries..."
	rm -f sdk/nhp-agent.so sdk/nhp-agent.dylib sdk/nhp-agent.dll sdk/nhp-agent.h

.PHONY: all generate-version-and-build init build build-sdk build-sdk-linux build-sdk-macos build-sdk-windows clean-sdk
