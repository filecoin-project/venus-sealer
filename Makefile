deps:
	git submodule update --init
	./extern/filecoin-ffi/install-filcrypto
build:
	go build -o venus-sealer ./app/venus-sealer
	go build -o venus-worker ./app/venus-worker