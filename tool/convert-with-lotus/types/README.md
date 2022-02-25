All types in this package fetched from lotus@v1.14.0-rc4.

We should go mod init 'convert-with-lotus' director, ref the `SectorInfo`(and it's member fields types)  defined in lotus directly.

but because of following reason:

the lotus@v1.14.0 use a package: `github.com/ipfs/go-datastore@v0.4.6`
which doesn't compatible with the version(v0.5.1) referenced by venus-sealer.

we HAVE TO use this temporary way to use the type: `SectorInfo`(and it's member fields types) defined in louts.
