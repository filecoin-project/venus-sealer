package types

type CallState uint64

const (
	CallStarted CallState = iota
	CallDone
	// returned -> remove
)

type Call struct {
	ID      CallID
	RetType ReturnType

	State CallState

	Result *ManyBytes // json bytes
}
