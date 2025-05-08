package typecase

// State machine for handling split algorithm.
type state uint8

const (
	stateNone state = iota
	stateLower
	stateFirstUpper
	stateUpper
	stateSymbol
)
