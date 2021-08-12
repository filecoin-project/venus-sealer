# Venus Sealer

This project is an independent module(mining system) supporting venus, which is currently under development actively by venus team and community members,
***please consider this carefully before deploy it  in the production environment***, we do not guarantee compatibility.

## How to Build

```sh
    make deps
    make build
```

## Run a local net

### init miner 
```shell script
./venus-sealer init \
--worker <WORKER_ADDRESS> \
--owner <OWNER_ADDRESS>  \
# Choose between 32G or 64G for mainnet
--sector-size <sector size> \
# Choose from nerpa, calibration for testnets
# Leave out this flag for mainnet
--network <network type> \
# Config for different shared venus modules
--node-url /ip4/<IP_ADDRESS_OF_VENUS>/tcp/3453 \
--messager-url /ip4/<IP_ADDRESS_OF_VENUS_MESSAGER>/tcp/<PORT_OF_VENUS_MESSAGER> \
--gateway-url /ip4/<IP_ADDRESS_OF_VENUS_GATEWAY>/tcp/<PORT_OF_VENUS_GATEWAY> \
--auth-token <AUTH_TOKEN_FOR_ACCOUNT_NAME> \
# Flags sealer to not storing any sealed sectors on the machine it runs on
# You can leave out this flag if you are on testnet
--no-local-storage
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

