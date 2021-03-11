package types

type ReturnType string

const (
	ReturnAddPiece        ReturnType = "ReturnAddPiece"
	ReturnSealPreCommit1  ReturnType = "ReturnSealPreCommit1"
	ReturnSealPreCommit2  ReturnType = "ReturnSealPreCommit2"
	ReturnSealCommit1     ReturnType = "ReturnSealCommit1"
	ReturnSealCommit2     ReturnType = "ReturnSealCommit2"
	ReturnFinalizeSector  ReturnType = "ReturnFinalizeSector"
	ReturnReleaseUnsealed ReturnType = "ReturnReleaseUnsealed"
	ReturnMoveStorage     ReturnType = "ReturnMoveStorage"
	ReturnUnsealPiece     ReturnType = "ReturnUnsealPiece"
	ReturnReadPiece       ReturnType = "ReturnReadPiece"
	ReturnFetch           ReturnType = "ReturnFetch"
)
