package tlogsql

import (
	"context"
	"database/sql"

	"github.com/opessa/tlog-pipeline/internal/naming"
	"github.com/opessa/tlog-pipeline/internal/tlog"
	"github.com/opessa/tlog-pipeline/internal/tlog/common"
)

// TransferGenerator: TLOG_INVENTORY_TRANSFER.
//
// La tabla driver LAGERBEW no está en el schema SQLite (ni en los CSVs del
// export). Hasta que sea provista, este generator devuelve Empty=true.
// Para implementar: agregar LAGERBEW en internal/sqldb/schema.go (DDL +
// allSchemas) y luego escribir las queries equivalentes acá.
type TransferGenerator struct{}

func (TransferGenerator) Type() naming.TLOGType { return naming.TLOGTransfer }

func (TransferGenerator) ListCandidateIDs(_ context.Context, _ *sql.DB, _ string) ([]string, error) {
	return nil, nil
}

func (TransferGenerator) Generate(_ context.Context, _ *sql.DB, _ *common.HeaderCtx, _ string, _ tlog.DocSeqMap, _ tlog.DocSeqMap, _ int) (*tlog.GenerateResult, error) {
	return &tlog.GenerateResult{Empty: true}, nil
}
