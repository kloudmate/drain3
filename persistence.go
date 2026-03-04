package drain3

// PersistenceHandler defines the interface for saving/loading Drain state.
type PersistenceHandler interface {
	SaveState(state []byte) error
	LoadState() ([]byte, error)
}

// StateClearer is an optional interface for persistence backends that can
// clear corrupted state and continue with a fresh miner state.
type StateClearer interface {
	ClearState() error
}
