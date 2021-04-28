# Venus Sealer

This project is a mining system supporting venus, is currently under active development,
***please do not use in the production environment***, we do not guarantee compatibility.

## How to Build

```sh
    make deps
    make lint
    make build
```

## Run a local net

### init miner 
```shell script
 ./venus-sealer init \
 --worker <bls address 1> \
 --owner <bls address 2>  \
 --sector-size <sector size> \
 --network <network type> \
 --node-url /ip4/<IP3>/tcp/3453 \
 --node-token <auth token sealer> \
 --messager-url http://<IP4>:39812/rpc/v0 \
 --no-local-storage \
 --messager-token <auth token sealer> \
 --wallet-name testminer                
```
### run miner

```shell script
    ./venus-sealer run
```

### Command

The command line is the same as lotus-miner, but note that the commands related to deal is removed, and this part will be implemented in another tool

```shell script
    ./venus-sealer info               # show miner infomation
    ./venus-sealer sectors pledge     # do a pledge sector
    ./venus-sealer sectors list       # show local sectors status
    ./venus-sealer secctors stats 1   # show infomation of sector 1
```

