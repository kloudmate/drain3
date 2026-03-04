package drain3

// MemoryPersistence is an in-memory PersistenceHandler, useful for testing.
type MemoryPersistence struct {
	data []byte
}

// NewMemoryPersistence creates a new in-memory persistence handler.
func NewMemoryPersistence() *MemoryPersistence {
	return &MemoryPersistence{}
}

// SaveState stores state in memory.
func (m *MemoryPersistence) SaveState(state []byte) error {
	m.data = make([]byte, len(state))
	copy(m.data, state)
	return nil
}

// LoadState returns the previously saved state, or nil if none.
func (m *MemoryPersistence) LoadState() ([]byte, error) {
	if m.data == nil {
		return nil, nil
	}
	out := make([]byte, len(m.data))
	copy(out, m.data)
	return out, nil
}
