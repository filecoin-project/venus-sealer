SHELL=/usr/bin/env bash

CLEAN:=
BINS:=

ldflags=-X=github.com/filecoin-project/venus-sealer/constants.CurrentCommit=+git.$(subst -,.,$(shell git describe --always --match=NeVeRmAtCh --dirty 2>/dev/null || git rev-parse --short HEAD 2>/dev/null))
ifneq ($(strip $(LDFLAGS)),)
	ldflags+=-extldflags=$(LDFLAGS)
endif

GOFLAGS+=-ldflags="$(ldflags)"

## FFI

FFI_PATH:=extern/filecoin-ffi/

CLEAN+=build/.filecoin-install

build: builtin-actor-bundles
	go build $(GOFLAGS) -o venus-sealer ./app/venus-sealer
	go build $(GOFLAGS) -o venus-worker ./app/venus-worker
	BINS+=venus-sealer
	BINS+=venus-worker

deps:
	git submodule update --init
	./extern/filecoin-ffi/install-filcrypto

# builtin actor bundles
builtin-actor-bundles:
	./builtin-actors/fetch-bundles.sh

# TOOLS

api-docs-gen:
	go run ./tool/api-docs-gen/cmd ./api/storage_struct.go StorageMiner api ./api/ > ./docs/api-documents.md


lotus-fix:
	rm -f lotus-fix
	go build -o lotus-fix ./tool/convert-with-lotus/to-lotus

.PHONY: lotus-fix
BINS+=lotus-fix

lotus-convert:
	rm -f lotus-convert
	go build -o lotus-convert ./tool/convert-with-lotus/from-lotus

.PHONY: lotus-convert
BINS+=lotus-convert

lint:
	go run github.com/golangci/golangci-lgit int/cmd/golangci-lint run

clean:
	rm -rf $(CLEAN) $(BINS)
	-$(MAKE) -C $(FFI_PATH) clean
.PHONY: clean
