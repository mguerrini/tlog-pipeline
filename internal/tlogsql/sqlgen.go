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
	"time"

	"github.com/opessa/tlog-pipeline/internal/naming"
	"github.com/opessa/tlog-pipeline/internal/sequence"
	"github.com/opessa/tlog-pipeline/internal/tlog"
	"github.com/opessa/tlog-pipeline/internal/tlog/common"
)

// Generator es la interfaz de los generators SQL.
//
// El flujo de uso tiene dos fases por KST:
//  1. BuildSeqMap: pre-asigna SequenceNumbers (docID → seqNum) para los
//     candidatos del KST antes de escribir cualquier archivo. Retorna nil para
//     Cierre (no participa en pre-asignación).
//  2. Generate: produce los XMLs usando los seqNums del DocSeqMap.
//     Para Cierre, seqMap es nil y se usa startCounter.
type Generator interface {
	Type() naming.TLOGType

	// BuildSeqMap pre-asigna SequenceNumbers para los candidatos del KST.
	// startCounter es el próximo índice global para este tipo de documento.
	// Retorna nil si el generator no usa pre-asignación (Cierre).
	// El segundo valor es la cantidad de seqNums consumidos (puede diferir de
	// len(seqMap) cuando varios IDs comparten el mismo seqNum, como en FiscalDoc).
	BuildSeqMap(ctx context.Context, conn *sql.DB, kstID string, businessDay time.Time, startCounter int) (tlog.DocSeqMap, int, error)

	// Generate produce los archivos XML del KST. seqMap contiene los seqNums
	// pre-asignados por BuildSeqMap (nil para Cierre, que usa startCounter).
	// crossSeqMap contiene los seqNums del documento "par"; nil si no aplica.
	Generate(ctx context.Context, conn *sql.DB, h *common.HeaderCtx, kstID string, seqMap tlog.DocSeqMap, crossSeqMap tlog.DocSeqMap, startCounter int) (*tlog.GenerateResult, error)
}

// buildSeqMapFromIDs construye el DocSeqMap asignando un SEQUENCENUMBER a
// cada ID comenzando en startCounter. Devuelve el map y la cantidad de
// seqNums consumidos (= len(ids)).
func buildSeqMapFromIDs(ids []string, businessDay time.Time, doc sequence.DocumentNumber, startCounter int) (tlog.DocSeqMap, int, error) {
	if len(ids) == 0 {
		return nil, 0, nil
	}
	sm := make(tlog.DocSeqMap, len(ids))
	for i, id := range ids {
		seqNum, err := sequence.Build(businessDay, doc, startCounter+i)
		if err != nil {
			return nil, 0, err
		}
		sm[id] = seqNum
	}
	return sm, len(ids), nil
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