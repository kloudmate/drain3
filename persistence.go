package drain3

// PersistenceHandler defines the interface for saving/loading Drain state.
type PersistenceHandler interface {
	SaveState(state []byte) error
	LoadState() ([]byte, error)
}
