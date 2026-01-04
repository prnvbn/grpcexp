.PHONY : dist windows darwin-amd64 darwin-arm64 linux-arm64 linux-amd64 clean

DIST_DIR="dist"
ZIP="zip -m"

LDFLAGS="-s -w"
BUILD_FLAGS=-ldflags=$(LDFLAGS)

define build-target
CGO_ENABLED=0 GOOS=$(1)$(if $(2), GOARCH=$(2)) \
	go build $(BUILD_FLAGS) -o $(DIST_DIR)/grpcexp-$(3) ./cmd/grpcexp
endef

dist:
	mkdir -p $(DIST_DIR)
	$(MAKE) windows darwin-arm64 darwin-amd64 linux-arm64 linux-amd64

windows:
	$(call build-target,windows,,windows.exe)

darwin-amd64:
	$(call build-target,darwin,amd64,darwin-amd64)
darwin-arm64:
	$(call build-target,darwin,arm64,darwin-arm64)

linux-arm64:
	$(call build-target,linux,arm64,linux-arm64)

linux-amd64:
	$(call build-target,linux,amd64,linux-amd64)

clean:
	rm -r $(DIST_DIR)/