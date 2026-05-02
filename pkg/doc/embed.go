package doc

import (
	"embed"

	"github.com/go-go-golems/glazed/pkg/help"
)

//go:embed *.md
var docFS embed.FS

// AddDocToHelpSystem loads codebase-browser help entries into the Glazed help system.
func AddDocToHelpSystem(helpSystem *help.HelpSystem) error {
	return helpSystem.LoadSectionsFromFS(docFS, ".")
}
