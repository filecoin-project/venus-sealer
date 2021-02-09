# Venus Sealer

This project is a mining system supporting venus, is currently under active development,
***please do not use in the production environment***, we do not guarantee compatibility.

## How to Build

```sh
    make deps
    make build
```

## Run a local net

init miner 
```shell script
    ./sealer init --genesis-miner --actor=f01000 --sector-size=2KiB --pre-sealed-sectors=~/.genesis-sectors --pre-sealed-metadata=~/.genesis-sectors/pre-seal-t01000.json --network 2k --nosync # for 2k devnet genesis miner
    ./sealer init --owner xx --worker xx --network calibration --nosync   #for calibration common miner
    ./sealer init --owner xx --worker xx --network mainnet --nosync       #for mainnet common miner
    ./sealer init --owner xx --worker xx  --nosync                        #for mainnet common miner, mainnet is default network type
```
run miner

```shell script
    ./sealer run --nosync
```


show miner info

```shell script
    ./sealer info               # show miner infomation
    ./sealer sectors pledge     # do a pledge sector
    ./sealer sectors list       # show local sectors status
    ./sealer secctors stats 1   # show infomation of sector 1
```
    

