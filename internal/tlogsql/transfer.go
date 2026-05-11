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

func (TransferGenerator) Generate(ctx context.Context, conn *sql.DB, h *common.HeaderCtx, kstID string, startCounter int) (*tlog.GenerateResult, error) {
	_ = ctx
	_ = conn
	_ = h
	_ = kstID
	_ = startCounter
	return &tlog.GenerateResult{Empty: true}, nil
}
