APP_NAME := mssh
APP_ID := io.github.sf966gpm.mssh
GO := go
FYNE_CROSS ?= fyne-cross
FYNE_CROSS_ENV ?= GOTOOLCHAIN=auto
APP_PACKAGE := ./cmd/$(APP_NAME)
WINDOWS_ARCH ?= amd64
MACOS_ARCH ?= arm64,amd64
MACOS_SDK_PATH ?=

.PHONY: build build-windows build-macos build-all run

build:
	mkdir -p bin
	$(GO) build -o bin/$(APP_NAME) $(APP_PACKAGE)

# Cross builds require fyne-cross and Docker. Set MACOS_SDK_PATH when building
# macOS from Linux or Windows.
build-windows:
	$(FYNE_CROSS) windows -arch=$(WINDOWS_ARCH) -app-id $(APP_ID) -env=$(FYNE_CROSS_ENV) -output $(APP_NAME) $(APP_PACKAGE)

build-macos:
	$(FYNE_CROSS) darwin -arch=$(MACOS_ARCH) -app-id $(APP_ID) -env=$(FYNE_CROSS_ENV) $(if $(MACOS_SDK_PATH),-macosx-sdk-path="$(MACOS_SDK_PATH)") -output $(APP_NAME) $(APP_PACKAGE)

build-all: build build-windows build-macos

run:
	$(GO) run $(APP_PACKAGE)
