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
    # for 2k devnet genesis miner
    ./venus-sealer init --genesis-miner --actor=f01000 --sector-size=2KiB --pre-sealed-sectors=~/.genesis-sectors --pre-sealed-metadata=~/.genesis-sectors/pre-seal-t01000.json --network 2k --nosync
    
    # for calibration common miner
    ./venus-sealer init --owner xx --worker xx --network calibration --nosync   

    # for mainnet common miner
    ./venus-sealer init --owner xx --worker xx --network mainnet --nosync
    
    # for mainnet common miner, mainnet is default network type
    ./venus-sealer init --owner xx --worker xx  --nosync                       
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

