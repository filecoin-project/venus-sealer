SHELL=/usr/bin/env bash

CLEAN:=
BINS:=

github.com/filecoin-project/venus-sealer/build.CurrentCommit=+git.$(subst -,.,$(shell git describe --always --match=NeVeRmAtCh --dirty 2>/dev/null || git rev-parse --short HEAD 2>/dev/null))

## FFI

FFI_PATH:=extern/filecoin-ffi/

CLEAN+=build/.filecoin-install

build:
	go build -o venus-sealer ./app/venus-sealer
	go build -o venus-worker ./app/venus-worker
	BINS+=venus-sealer
	BINS+=venus-worker

deps:
	git submodule update --init
	./extern/filecoin-ffi/install-filcrypto

lint:
	go run github.com/golangci/golangci-lint/cmd/golangci-lint run

clean:
	rm -rf $(CLEAN) $(BINS)
	-$(MAKE) -C $(FFI_PATH) clean
.PHONY: clean
