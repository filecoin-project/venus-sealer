package repo

type RepoType int

const (
	_                 = iota // Default is invalid
	FullNode RepoType = iota
	StorageMiner
	Worker
	Wallet
)
