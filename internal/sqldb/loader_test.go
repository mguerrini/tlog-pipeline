package sqldb

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/opessa/tlog-pipeline/internal/csvio"
)

// TestLoadSampleData verifica que el DDL generado y la carga funcionan
// contra los CSVs sample, y que todas las columnas del CSV quedan en la
// tabla SQLite.
func TestLoadSampleData(t *testing.T) {
	srcDir := filepath.Join("..", "..", "sample_data", "20260505")
	dbPath := filepath.Join(t.TempDir(), "pipeline.db")

	res, err := Load(srcDir, dbPath, ",")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, s := range res.Stats {
		if s.Err != nil {
			t.Errorf("tabla %s falló: %v", s.Table, s.Err)
		}
	}

	// Spot-check: contar columnas en cada tabla y comparar con el largo
	// de la lista del schema.
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer conn.Close()

	for _, ts := range allSchemas() {
		want := len(ts.cols)
		var got int
		row := conn.QueryRow(`SELECT COUNT(*) FROM pragma_table_info(?)`, ts.sqliteName)
		if err := row.Scan(&got); err != nil {
			t.Fatalf("pragma_table_info %s: %v", ts.sqliteName, err)
		}
		if got != want {
			t.Errorf("tabla %s: got %d columnas, want %d", ts.sqliteName, got, want)
		}
	}

	// Y que LF_VERT (la columna que el generator de reception usaba pero
	// no existía en el viejo schema) haya quedado consultable.
	var n int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM LIEFER WHERE LF_VERT IS NOT NULL`).Scan(&n); err != nil {
		t.Fatalf("query LF_VERT: %v", err)
	}
	t.Logf("LIEFER.LF_VERT no-null rows: %d", n)

	// Comparar columnas del CSV con las declaradas: no debe faltar ninguna
	// del CSV en el schema.
	csvFiles, err := csvio.FindFiles(srcDir)
	if err != nil {
		t.Fatalf("FindFiles: %v", err)
	}
	for _, ts := range allSchemas() {
		key := ts.sqliteName
		path, ok := csvFiles[key]
		if !ok {
			t.Errorf("CSV no encontrado para %s", ts.sqliteName)
			continue
		}
		header, _, err := csvio.Read(path, ",")
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		schemaSet := make(map[string]struct{}, len(ts.cols))
		for _, c := range ts.cols {
			schemaSet[c.name] = struct{}{}
		}
		for _, h := range header {
			if _, ok := schemaSet[h]; !ok {
				t.Errorf("tabla %s: columna CSV %q no está en el schema", ts.sqliteName, h)
			}
		}
	}
}