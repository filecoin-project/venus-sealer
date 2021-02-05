deps:
	./extern/filecoin-ffi/install-filcrypto

build:
	go build -o venus-sealer ./sealer