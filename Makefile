build:
	go build -o venus-sealer ./app/venus-sealer
	go build -o venus-worker ./app/venus-worker

deps:
	git submodule update --init
	./extern/filecoin-ffi/install-filcrypto

lint:
	go run github.com/golangci/golangci-lint/cmd/golangci-lint run

