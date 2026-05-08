package sqldb

import (
	"database/sql"
	"fmt"
	"strconv"

	_ "modernc.org/sqlite"

	"github.com/opessa/tlog-pipeline/internal/db"
)

// LoadStore abre la DB SQLite en dbPath y reconstruye un db.Store equivalente
// al que produce create_db a partir de los CSVs. Cada celda se devuelve como
// string (vacío para NULL) para mantener la semántica que esperan los generators.
func LoadStore(dbPath string) (*db.Store, error) {
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("abrir sqlite %s: %w", dbPath, err)
	}
	defer conn.Close()

	store := db.NewStore()
	for _, ts := range allSchemas() {
		header, rows, err := readTable(conn, ts)
		if err != nil {
			return nil, fmt.Errorf("leer tabla %s: %w", ts.sqliteName, err)
		}
		// Usar el nombre lógico (DAILYTOTALS) que esperan los generators y
		// el resto del pipeline. El schema SQLite usa DAILYTOTALS1.
		name := ts.sqliteName
		if name == "DAILYTOTALS1" {
			name = "DAILYTOTALS"
		}
		store.AddTable(name, header, rows)
	}
	store.BuildIndexes()
	return store, nil
}

func readTable(conn *sql.DB, ts *tableSchema) ([]string, []db.Row, error) {
	header := make([]string, len(ts.cols))
	quoted := make([]string, len(ts.cols))
	for i, c := range ts.cols {
		header[i] = c.name
		quoted[i] = `"` + c.name + `"`
	}

	q := fmt.Sprintf(`SELECT %s FROM "%s"`, joinCommas(quoted), ts.sqliteName)
	rs, err := conn.Query(q)
	if err != nil {
		return nil, nil, err
	}
	defer rs.Close()

	var out []db.Row
	scanArgs := make([]any, len(ts.cols))
	holders := make([]sql.NullString, len(ts.cols))
	for i := range scanArgs {
		scanArgs[i] = &holders[i]
	}

	for rs.Next() {
		if err := rs.Scan(scanArgs...); err != nil {
			return nil, nil, err
		}
		row := make(db.Row, len(ts.cols))
		for i, c := range ts.cols {
			if !holders[i].Valid {
				row[c.name] = ""
				continue
			}
			row[c.name] = formatVal(holders[i].String, c.typ)
		}
		out = append(out, row)
	}
	if err := rs.Err(); err != nil {
		return nil, nil, err
	}
	return header, out, nil
}

// formatVal normaliza el string que devuelve SQLite para que coincida con el
// formato que producía el CSV original (los generators dependen de esto).
func formatVal(s string, ct colType) string {
	switch ct {
	case colInteger:
		// SQLite puede devolver "1" o "1.0" según el driver — normalizar a entero.
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return strconv.FormatInt(i, 10)
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return strconv.FormatInt(int64(f), 10)
		}
		return s
	case colReal:
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return strconv.FormatFloat(f, 'f', -1, 64)
		}
		return s
	default:
		return s
	}
}

func joinCommas(xs []string) string {
	out := ""
	for i, x := range xs {
		if i > 0 {
			out += ", "
		}
		out += x
	}
	return out
}