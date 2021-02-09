deps:
	git submodule update --init
	./extern/filecoin-ffi/install-filcrypto
build:
	go build -o venus-sealer ./sealer