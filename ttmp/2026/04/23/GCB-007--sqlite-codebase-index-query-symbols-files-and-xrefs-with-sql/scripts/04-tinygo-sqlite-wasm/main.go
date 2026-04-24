//go:build wasm

// main.go is a TinyGo WASM module that provides Go-side query helpers
// for the SQLite codebase browser demo. It runs alongside sql.js in the
// browser — Go handles the "business logic" (parsing query results,
// formatting output) while sql.js handles the actual SQL execution.
//
// Exports on window.goQuery:
//   - hello()                  → verify WASM loaded
//   - getSchemaSQL()           → CREATE TABLE statements for the demo
//   - getInsertSQL()           → INSERT statements with sample data
//   - formatResults(json)      → pretty-print query results as HTML table
//   - suggestQueries()         → list of interesting SQL queries to try
package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"syscall/js"
)

func main() {
	exports := js.ValueOf(map[string]interface{}{})
	keepAlive := []js.Func{}

	register := func(name string, fn func(this js.Value, args []js.Value) interface{}) {
		f := js.FuncOf(fn)
		keepAlive = append(keepAlive, f)
		exports.Set(name, f)
	}

	register("hello", func(this js.Value, args []js.Value) interface{} {
		return "go-query-wasm ready"
	})

	register("getSchemaSQL", func(this js.Value, args []js.Value) interface{} {
		return schemaSQL
	})

	register("getInsertSQL", func(this js.Value, args []js.Value) interface{} {
		return insertSQL
	})

	register("formatResults", func(this js.Value, args []js.Value) interface{} {
		if len(args) < 1 {
			return "<p>no data</p>"
		}
		return formatAsHTMLTable(args[0].String())
	})

	register("suggestQueries", func(this js.Value, args []js.Value) interface{} {
		data, _ := json.Marshal(suggestedQueries)
		return string(data)
	})

	js.Global().Set("goQuery", exports)
	<-make(chan struct{})
}

func formatAsHTMLTable(jsonStr string) string {
	var rows []map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &rows); err != nil {
		return fmt.Sprintf("<p class='error'>Parse error: %s</p>", err)
	}
	if len(rows) == 0 {
		return "<p>No results</p>"
	}

	var sb strings.Builder
	sb.WriteString("<table><thead><tr>")
	// Header from first row
	for k := range rows[0] {
		sb.WriteString(fmt.Sprintf("<th>%s</th>", k))
	}
	sb.WriteString("</tr></thead><tbody>")
	for _, row := range rows {
		sb.WriteString("<tr>")
		for _, v := range row {
			sb.WriteString(fmt.Sprintf("<td>%v</td>", v))
		}
		sb.WriteString("</tr>")
	}
	sb.WriteString("</tbody></table>")
	return sb.String()
}

var suggestedQueries = []struct {
	Title string `json:"title"`
	SQL   string `json:"sql"`
}{
	{
		"All packages",
		"SELECT name, import_path FROM packages ORDER BY import_path",
	},
	{
		"Symbol count by kind",
		"SELECT kind, COUNT(*) as count FROM symbols GROUP BY kind ORDER BY count DESC",
	},
	{
		"Exported functions with doc",
		"SELECT name, signature, doc FROM symbols WHERE kind='func' AND exported=1 AND doc != '' LIMIT 10",
	},
	{
		"Full-text search: 'handle'",
		"SELECT s.name, s.kind, p.import_path, bm25(symbols_fts) as rank FROM symbols_fts fts JOIN symbols s ON s.rowid=fts.rowid JOIN packages p ON p.id=s.package_id WHERE symbols_fts MATCH 'handle' ORDER BY rank LIMIT 10",
	},
	{
		"Full-text search: 'search'",
		"SELECT s.name, s.kind, p.import_path, bm25(symbols_fts) as rank FROM symbols_fts fts JOIN symbols s ON s.rowid=fts.rowid JOIN packages p ON p.id=s.package_id WHERE symbols_fts MATCH 'search' ORDER BY rank LIMIT 10",
	},
	{
		"Most referenced symbols (top 10)",
		"SELECT s.name, s.kind, p.import_path, COUNT(*) as refs FROM refs r JOIN symbols s ON s.id=r.to_id JOIN packages p ON p.id=s.package_id GROUP BY s.id ORDER BY refs DESC LIMIT 10",
	},
	{
		"Symbols per package",
		"SELECT p.name, p.import_path, COUNT(s.id) as symbol_count FROM packages p LEFT JOIN symbols s ON s.package_id=p.id GROUP BY p.id ORDER BY symbol_count DESC",
	},
	{
		"Exported symbols without docs",
		"SELECT s.name, s.kind, p.import_path FROM symbols s JOIN packages p ON p.id=s.package_id WHERE s.exported=1 AND (s.doc IS NULL OR s.doc='') ORDER BY p.import_path",
	},
}

const schemaSQL = `
CREATE TABLE IF NOT EXISTS packages (
    id TEXT PRIMARY KEY, import_path TEXT NOT NULL, name TEXT NOT NULL,
    doc TEXT DEFAULT '', language TEXT DEFAULT 'go'
);
CREATE TABLE IF NOT EXISTS files (
    id TEXT PRIMARY KEY, path TEXT NOT NULL, package_id TEXT NOT NULL REFERENCES packages(id),
    language TEXT DEFAULT 'go', size INTEGER DEFAULT 0, line_count INTEGER DEFAULT 0
);
CREATE TABLE IF NOT EXISTS symbols (
    id TEXT PRIMARY KEY, kind TEXT NOT NULL, name TEXT NOT NULL,
    package_id TEXT NOT NULL REFERENCES packages(id),
    file_id TEXT NOT NULL REFERENCES files(id),
    start_line INTEGER NOT NULL, end_line INTEGER DEFAULT 0,
    signature TEXT DEFAULT '', doc TEXT DEFAULT '',
    receiver TEXT DEFAULT NULL, exported BOOLEAN DEFAULT FALSE, language TEXT DEFAULT 'go'
);
CREATE TABLE IF NOT EXISTS refs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    from_id TEXT NOT NULL REFERENCES symbols(id),
    to_id TEXT NOT NULL REFERENCES symbols(id),
    kind TEXT DEFAULT '', file_id TEXT NOT NULL REFERENCES files(id),
    start_line INTEGER NOT NULL, end_line INTEGER DEFAULT 0
);
CREATE TABLE IF NOT EXISTS index_meta (
    id INTEGER PRIMARY KEY CHECK (id=1),
    version TEXT DEFAULT '1', generated_at TEXT NOT NULL,
    module TEXT DEFAULT '', go_version TEXT DEFAULT ''
);
CREATE VIRTUAL TABLE IF NOT EXISTS symbols_fts USING fts5(
    name, signature, doc, kind, import_path,
    content='symbols', content_rowid='rowid'
);
CREATE TRIGGER IF NOT EXISTS symbols_fts_ins AFTER INSERT ON symbols BEGIN
    INSERT INTO symbols_fts(rowid,name,signature,doc,kind,import_path)
    SELECT new.rowid,new.name,new.signature,new.doc,new.kind,p.import_path
    FROM packages p WHERE p.id=new.package_id;
END;
CREATE INDEX IF NOT EXISTS idx_symbols_name ON symbols(name);
CREATE INDEX IF NOT EXISTS idx_symbols_kind ON symbols(kind);
CREATE INDEX IF NOT EXISTS idx_refs_from ON refs(from_id);
CREATE INDEX IF NOT EXISTS idx_refs_to ON refs(to_id);
`

const insertSQL = `
INSERT INTO index_meta VALUES (1,'1','2026-04-23T12:00:00Z','github.com/example/demo','go1.25');

INSERT INTO packages VALUES ('pkg:main','github.com/example/demo','main','','go');
INSERT INTO packages VALUES ('pkg:server','github.com/example/demo/server','server','Package server provides an HTTP server.','go');
INSERT INTO packages VALUES ('pkg:indexer','github.com/example/demo/indexer','indexer','Package indexer extracts symbols from Go source.','go');

INSERT INTO files VALUES ('file:main.go','main.go','pkg:main','go',1200,45);
INSERT INTO files VALUES ('file:server.go','server/server.go','pkg:server','go',3400,98);
INSERT INTO files VALUES ('file:handler.go','server/handler.go','pkg:server','go',2100,67);
INSERT INTO files VALUES ('file:indexer.go','indexer/indexer.go','pkg:indexer','go',5600,180);

INSERT INTO symbols VALUES ('sym:main.func.main','func','main','pkg:main','file:main.go',12,25,'func()','','',1,'go');
INSERT INTO symbols VALUES ('sym:server.func.New','func','New','pkg:server','file:server.go',18,35,'func(cfg Config) *Server','New creates a new HTTP server with the given configuration.','','',1,'go');
INSERT INTO symbols VALUES ('sym:server.func.Start','func','Start','pkg:server','file:server.go',37,80,'func(s *Server) Start(addr string) error','Start begins listening on the given address. It blocks until the server exits.','Server',0,'go');
INSERT INTO symbols VALUES ('sym:server.func.Shutdown','func','Shutdown','pkg:server','file:server.go',82,95,'func(s *Server) Shutdown(ctx context.Context) error','Shutdown gracefully shuts down the server.','Server',0,'go');
INSERT INTO symbols VALUES ('sym:server.func.handleSearch','func','handleSearch','pkg:server','file:handler.go',10,45,'func(s *Server, w http.ResponseWriter, r *http.Request)','handleSearch processes search queries against the symbol index.','Server',0,'go');
INSERT INTO symbols VALUES ('sym:server.func.handleSymbol','func','handleSymbol','pkg:server','file:handler.go',47,72,'func(s *Server, w http.ResponseWriter, r *http.Request)','handleSymbol returns details for a single symbol.','Server',0,'go');
INSERT INTO symbols VALUES ('sym:server.type.Server','type','Server','pkg:server','file:server.go',5,16,'struct{...}','Server is the main HTTP server for the codebase browser.','','',1,'go');
INSERT INTO symbols VALUES ('sym:server.type.Config','type','Config','pkg:server','file:server.go',1,4,'struct{Addr string; Debug bool}','Config holds server configuration.','','',1,'go');
INSERT INTO symbols VALUES ('sym:indexer.func.Extract','func','Extract','pkg:indexer','file:indexer.go',15,120,'func(dir string) (*Index, error)','Extract walks the Go source tree and produces a symbol index.','','',1,'go');
INSERT INTO symbols VALUES ('sym:indexer.func.parseFile','func','parseFile','pkg:indexer','file:indexer.go',122,170,'func(path string) ([]Symbol, error)','','','',0,'go');
INSERT INTO symbols VALUES ('sym:indexer.type.Index','type','Index','pkg:indexer','file:indexer.go',1,14,'struct{Packages []Package; Symbols []Symbol}','Index holds the complete extracted codebase index.','','',1,'go');

INSERT INTO refs VALUES (null,'sym:main.func.main','sym:server.func.New','call','file:main.go',15,17);
INSERT INTO refs VALUES (null,'sym:main.func.main','sym:server.func.Start','call','file:main.go',20,22);
INSERT INTO refs VALUES (null,'sym:server.func.Start','sym:server.func.handleSearch','call','file:server.go',50,52);
INSERT INTO refs VALUES (null,'sym:server.func.Start','sym:server.func.handleSymbol','call','file:server.go',53,55);
INSERT INTO refs VALUES (null,'sym:indexer.func.Extract','sym:indexer.func.parseFile','call','file:indexer.go',30,32);
INSERT INTO refs VALUES (null,'sym:server.func.New','sym:server.type.Config','use','file:server.go',19,21);
INSERT INTO refs VALUES (null,'sym:server.func.New','sym:server.type.Server','use','file:server.go',22,24);
INSERT INTO refs VALUES (null,'sym:indexer.func.Extract','sym:indexer.type.Index','use','file:indexer.go',16,18);
`
