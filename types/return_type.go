package types

type ReturnType string

const (
	ReturnDataCid               ReturnType = "ReturnDataCid"
	ReturnAddPiece              ReturnType = "ReturnAddPiece"
	ReturnSealPreCommit1        ReturnType = "ReturnSealPreCommit1"
	ReturnSealPreCommit2        ReturnType = "ReturnSealPreCommit2"
	ReturnSealCommit1           ReturnType = "ReturnSealCommit1"
	ReturnSealCommit2           ReturnType = "ReturnSealCommit2"
	ReturnFinalizeSector        ReturnType = "ReturnFinalizeSector"
	ReturnFinalizeReplicaUpdate ReturnType = "ReturnFinalizeReplicaUpdate"
	ReturnReplicaUpdate         ReturnType = "ReturnReplicaUpdate"
	ReturnProveReplicaUpdate1   ReturnType = "ReturnProveReplicaUpdate1"
	ReturnProveReplicaUpdate2   ReturnType = "ReturnProveReplicaUpdate2"
	ReturnGenerateSectorKey     ReturnType = "ReturnGenerateSectorKey"
	ReturnReleaseUnsealed       ReturnType = "ReturnReleaseUnsealed"
	ReturnMoveStorage           ReturnType = "ReturnMoveStorage"
	ReturnUnsealPiece           ReturnType = "ReturnUnsealPiece"
	ReturnFetch                 ReturnType = "ReturnFetch"
)
