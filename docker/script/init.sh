#!/bin/sh

echo $@
./venus-sealer $@
./venus-sealer run
