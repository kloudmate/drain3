package drain3

import (
	"os"
)

// FilePersistence saves/loads Drain state to/from a file.
type FilePersistence struct {
	FilePath string
}

// NewFilePersistence creates a new file-based persistence handler.
func NewFilePersistence(filePath string) *FilePersistence {
	return &FilePersistence{FilePath: filePath}
}

// SaveState writes the state to the file, atomically.
func (f *FilePersistence) SaveState(state []byte) error {
	tmpPath := f.FilePath + ".tmp"
	if err := os.WriteFile(tmpPath, state, 0644); err != nil {
		return err
	}
	return os.Rename(tmpPath, f.FilePath)
}

// LoadState reads the state from the file.
// Returns nil, nil if the file does not exist.
func (f *FilePersistence) LoadState() ([]byte, error) {
	data, err := os.ReadFile(f.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return data, nil
}

// ClearState removes the persisted state file. Missing files are ignored.
func (f *FilePersistence) ClearState() error {
	if err := os.Remove(f.FilePath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
