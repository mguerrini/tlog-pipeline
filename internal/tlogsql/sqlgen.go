// Package tlogsql provee un set paralelo de generators que producen los XML
// TLOG ejecutando queries SQL contra la DB SQLite creada por create_sql_db.
//
// Decisión de diseño:
//   - Cada generator es 100% query-first: si falta un dato, se corrige el SQL
//     (o se agrega la columna en internal/sqldb/schema.go), no la lógica Go.
//   - Las queries devuelven map[string]string (NULL → ""), de modo que la
//     lógica de parsing reusa los mismos helpers (db.AsInt, db.AsFloat) que
//     consumen los generators in-memory en internal/tlog/*.
//   - La versión "store" original (internal/tlog) sigue siendo la que corre
//     cuando create_db.sql = false.
package tlogsql

import (
	"context"
	"database/sql"

	"github.com/opessa/tlog-pipeline/internal/naming"
	"github.com/opessa/tlog-pipeline/internal/tlog"
	"github.com/opessa/tlog-pipeline/internal/tlog/common"
)

// Generator es la interfaz de los generators SQL.
//
// El flujo de uso tiene dos fases por KST:
//  1. ListCandidateIDs: obtiene los IDs de los documentos fuente y permite
//     al step pre-asignar SequenceNumbers (docID → seqNum) antes de escribir
//     cualquier archivo. Cierre devuelve nil (no participa en pre-asignación).
//  2. Generate: produce los XMLs usando los seqNums del DocSeqMap.
//     Para Cierre, seqMap es nil y se usa startCounter.
type Generator interface {
	Type() naming.TLOGType

	// ListCandidateIDs devuelve los IDs fuente ordenados (LFS_ID, INV_ID,
	// VBR_ID, …) para pre-asignar SequenceNumbers antes de generar XMLs.
	// Devuelve nil para Cierre y Transfer (no participan en pre-asignación).
	ListCandidateIDs(ctx context.Context, conn *sql.DB, kstID string) ([]string, error)

	// Generate produce los archivos XML del KST. Para generators no-Cierre,
	// seqMap contiene los seqNums pre-asignados por ListCandidateIDs; en ese
	// caso startCounter se ignora. Para Cierre, seqMap es nil y se usa
	// startCounter.
	// crossSeqMap contiene los seqNums del documento "par" (p.ej. FiscalDocFC
	// para Reception y viceversa); nil si no aplica para el tipo de documento.
	Generate(ctx context.Context, conn *sql.DB, h *common.HeaderCtx, kstID string, seqMap tlog.DocSeqMap, crossSeqMap tlog.DocSeqMap, startCounter int) (*tlog.GenerateResult, error)
}

// queryRows ejecuta una query y devuelve cada fila como map[string]string.
// NULLs se convierten a "" para que los helpers de db.* (AsInt/AsFloat) los
// traten igual que las celdas vacías de los CSVs.
func queryRows(ctx context.Context, conn *sql.DB, query string, args ...any) ([]map[string]string, error) {
	rs, err := conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rs.Close()

	cols, err := rs.Columns()
	if err != nil {
		return nil, err
	}

	var out []map[string]string
	holders := make([]sql.NullString, len(cols))
	scanArgs := make([]any, len(cols))
	for i := range scanArgs {
		scanArgs[i] = &holders[i]
	}
	for rs.Next() {
		if err := rs.Scan(scanArgs...); err != nil {
			return nil, err
		}
		row := make(map[string]string, len(cols))
		for i, c := range cols {
			if holders[i].Valid {
				row[c] = holders[i].String
			} else {
				row[c] = ""
			}
		}
		out = append(out, row)
	}
	return out, rs.Err()
}

// selectOne devuelve la primera fila como map[string]string, o nil si no hay match.
func selectOne(ctx context.Context, conn *sql.DB, query string, args ...any) (map[string]string, error) {
	rows, err := queryRows(ctx, conn, query, args...)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

// fetchKostst busca el KOSTST por KST_ID. Si no existe devuelve un map vacío
// (mismo comportamiento que el lookup en memoria s.Kostst[kstID]).
func fetchKostst(ctx context.Context, conn *sql.DB, kstID string) (map[string]string, error) {
	row, err := selectOne(ctx, conn, `SELECT * FROM KOSTST WHERE KST_ID = ?`, kstID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return map[string]string{}, nil
	}
	return row, nil
}

// AllKstIDs lista todos los KST_ID de la tabla KOSTST. Útil para iterar
// retails desde el step. Devuelve también el KST_CODE para evitar otra query.
type Retail struct {
	KstID   string
	KstCode string
}

func AllRetails(ctx context.Context, conn *sql.DB) ([]Retail, error) {
	rows, err := queryRows(ctx, conn, `SELECT KST_ID, KST_CODE FROM KOSTST ORDER BY KST_ID`)
	if err != nil {
		return nil, err
	}
	out := make([]Retail, 0, len(rows))
	for _, r := range rows {
		out = append(out, Retail{KstID: r["KST_ID"], KstCode: r["KST_CODE"]})
	}
	return out, nil
}