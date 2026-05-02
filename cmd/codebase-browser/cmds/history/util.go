package history

import (
	"context"
	"fmt"
	"os"

	"github.com/go-go-golems/codebase-browser/internal/gitutil"
)

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	_, err = gitutil.ResolveRef(context.Background(), dir, "HEAD")
	if err != nil {
		return "", fmt.Errorf("%s is not a git repo: %w", dir, err)
	}
	return dir, nil
}
