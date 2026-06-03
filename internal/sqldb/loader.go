package sqldb

import (
	"database/sql"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/opessa/tlog-pipeline/internal/csvio"
)

// TableStat recoge el resultado de cargar una tabla.
type TableStat struct {
	Table    string
	CSVFile  string
	Inserted int
	Duration time.Duration
	Err      error
}

// LoadResult es el resultado completo de la carga de un día.
type LoadResult struct {
	DBPath    string
	Stats     []TableStat
	Orphans   []OrphanCheck
	OverallOK bool
}

// OrphanCheck describe el resultado de un chequeo de FK huérfana.
type OrphanCheck struct {
	Label        string
	OrphanRows   int64
	ExpectedZero bool
	OK           bool
}

// Load carga todos los CSVs de srcDir en una DB SQLite en dbPath usando sep
// como separador de campos (default "," si vacío). Devuelve el resultado con
// stats, conteos y chequeos de FK.
func Load(srcDir, dbPath, sep string) (*LoadResult, error) {
	if sep == "" {
		sep = ","
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("crear directorio para DB: %w", err)
	}
	// Borrar DB previa si existe (modo debug: siempre parte de cero)
	_ = os.Remove(dbPath)

	db, err := openDB(dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	if err := applyDDL(db, buildDDL()); err != nil {
		return nil, fmt.Errorf("aplicar DDL: %w", err)
	}

	result := &LoadResult{DBPath: dbPath, OverallOK: true}
	schemas := allSchemas()

	// Encontrar archivos CSV en srcDir
	csvFiles, err := csvio.FindFiles(srcDir)
	if err != nil {
		return nil, fmt.Errorf("buscar CSVs: %w", err)
	}
	fileByTable := make(map[string]string, len(csvFiles))
	for table, path := range csvFiles {
		fileByTable[table] = path
	}

	// Cargar en orden de dependencias
	for _, ts := range schemas {
		path, ok := fileByTable[ts.sqliteName]
		if !ok {
			result.Stats = append(result.Stats, TableStat{
				Table: ts.sqliteName, Err: fmt.Errorf("archivo CSV no encontrado"),
			})
			result.OverallOK = false
			continue
		}

		t0 := time.Now()
		header, rows, err := csvio.Read(path, sep)
		if err != nil {
			result.Stats = append(result.Stats, TableStat{
				Table: ts.sqliteName, CSVFile: path,
				Err: err, Duration: time.Since(t0),
			})
			result.OverallOK = false
			continue
		}

		inserted, err := bulkInsert(db, ts, header, rows)
		dur := time.Since(t0)
		result.Stats = append(result.Stats, TableStat{
			Table:    ts.sqliteName,
			CSVFile:  filepath.Base(path),
			Inserted: inserted,
			Duration: dur,
			Err:      err,
		})
		if err != nil {
			result.OverallOK = false
		}
	}

	// Índices
	_ = applyDDL(db, indexes) // no-fatal

	// Validaciones post-carga
	result.Orphans = runOrphanChecks(db)

	for _, o := range result.Orphans {
		if !o.OK {
			result.OverallOK = false
		}
	}

	return result, nil
}

// ── internals ──────────────────────────────────────────────────────────────

func openDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("abrir sqlite: %w", err)
	}
	for _, p := range []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA foreign_keys = OFF",
		"PRAGMA temp_store = MEMORY",
	} {
		if _, err := db.Exec(p); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("%s: %w", p, err)
		}
	}
	return db, nil
}

func applyDDL(db *sql.DB, src string) error {
	// Filtrar comentarios por línea antes de splitear por ";"
	lines := strings.Split(src, "\n")
	var clean []string
	for _, l := range lines {
		if t := strings.TrimSpace(l); strings.HasPrefix(t, "--") {
			continue
		}
		clean = append(clean, l)
	}
	for _, stmt := range strings.Split(strings.Join(clean, "\n"), ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if strings.HasPrefix(strings.ToUpper(stmt), "PRAGMA") {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("[%.80s]: %w", stmt, err)
		}
	}
	return nil
}

func bulkInsert(db *sql.DB, ts *tableSchema, header []string, rows []map[string]string) (int, error) {
	if len(rows) == 0 {
		return 0, nil
	}

	// Columnas efectivas: del header que existen en el schema
	schemaSet := make(map[string]struct{}, len(ts.cols))
	for _, c := range ts.cols {
		schemaSet[c.name] = struct{}{}
	}
	// Mantener orden del schema
	var effCols []string
	for _, c := range ts.cols {
		for _, h := range header {
			if h == c.name {
				effCols = append(effCols, c.name)
				break
			}
		}
	}
	if len(effCols) == 0 {
		return 0, fmt.Errorf("ninguna columna del schema encontrada en CSV para %s", ts.sqliteName)
	}

	quotedCols := make([]string, len(effCols))
	phs := make([]string, len(effCols))
	for i, c := range effCols {
		quotedCols[i] = `"` + c + `"`
		phs[i] = "?"
	}
	insertSQL := fmt.Sprintf(
		`INSERT OR IGNORE INTO "%s" (%s) VALUES (%s)`,
		ts.sqliteName,
		strings.Join(quotedCols, ", "),
		strings.Join(phs, ", "),
	)

	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	stmt, err := tx.Prepare(insertSQL)
	if err != nil {
		_ = tx.Rollback()
		return 0, fmt.Errorf("prepare %s: %w", ts.sqliteName, err)
	}

	args := make([]any, len(effCols))
	inserted := 0
	for _, row := range rows {
		for i, col := range effCols {
			args[i] = convertVal(row[col], ts.typeOf(col))
		}
		if _, err := stmt.Exec(args...); err != nil {
			_ = stmt.Close()
			_ = tx.Rollback()
			return inserted, fmt.Errorf("insert %s fila %d: %w", ts.sqliteName, inserted, err)
		}
		inserted++
	}
	_ = stmt.Close()
	return inserted, tx.Commit()
}

func convertVal(s string, ct colType) any {
	s = strings.TrimSpace(s)
	if s == "" || strings.EqualFold(s, "NULL") {
		return nil
	}
	switch ct {
	case colInteger:
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return i
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return int64(math.Round(f))
		}
		return nil
	case colReal:
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f
		}
		return nil
	default:
		return s
	}
}

func runOrphanChecks(db *sql.DB) []OrphanCheck {
	type def struct {
		label, child, chCol, parent, pCol string
		expectZero                         bool
	}
	defs := []def{
		{"LIEFERPOS.LFS_ID → LIEFERSCHEIN", "LIEFERPOS", "LFS_ID", "LIEFERSCHEIN", "LFS_ID", true},
		{"LIEFERPOS.ART_NR → ARTIKEL.ART_ID", "LIEFERPOS", "ART_NR", "ARTIKEL", "ART_ID", true},
		{"LIEFERSCHEIN.LF_ID → LIEFER", "LIEFERSCHEIN", "LF_ID", "LIEFER", "LF_ID", true},
		{"INVPOSART.INV_ID → INVENTUR (esperado)", "INVPOSART", "INV_ID", "INVENTUR", "INV_ID", false},
		{"INVPOSART.ART_ID → ARTIKEL", "INVPOSART", "ART_ID", "ARTIKEL", "ART_ID", true},
		{"INVPOSART.WGR_NR → WARENGRUPPE", "INVPOSART", "WGR_NR", "WARENGRUPPE", "WGR_ID", true},
		{"DAILYTOTALS.ART_ID → ARTIKEL", "DAILYTOTALS", "ART_ID", "ARTIKEL", "ART_ID", true},
		{"DAILYTOTALS.KST_ID → KOSTST", "DAILYTOTALS", "KST_ID", "KOSTST", "KST_ID", true},
		{"ARTIKEL.WGR_ID → WARENGRUPPE", "ARTIKEL", "WGR_ID", "WARENGRUPPE", "WGR_ID", true},
	}
	var out []OrphanCheck
	for _, d := range defs {
		q := fmt.Sprintf(
			`SELECT COUNT(*) FROM "%s" c LEFT JOIN "%s" p ON c."%s"=p."%s" WHERE c."%s" IS NOT NULL AND p."%s" IS NULL`,
			d.child, d.parent, d.chCol, d.pCol, d.chCol, d.pCol)
		var n int64
		_ = db.QueryRow(q).Scan(&n)
		ok := n == 0 || !d.expectZero
		out = append(out, OrphanCheck{
			Label: d.label, OrphanRows: n, ExpectedZero: d.expectZero, OK: ok,
		})
	}
	return out
}
