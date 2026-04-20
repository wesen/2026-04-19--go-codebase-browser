package indexer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Write serialises idx to path. If pretty is true the output is indented with
// two spaces, which makes diffs readable at the cost of file size.
func Write(idx *Index, path string, pretty bool) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	var (
		data []byte
		err  error
	)
	if pretty {
		data, err = json.MarshalIndent(idx, "", "  ")
	} else {
		data, err = json.Marshal(idx)
	}
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
