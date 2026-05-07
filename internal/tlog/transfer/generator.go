// Package transfer implementa el generador de TLOG_INVENTORY_TRANSFER.
//
// ESTADO: [UNKNOWN] — La tabla driver LAGERBEW no está incluida en los CSVs
// del export (no existe en el mapping de archivos del proyecto). Según
// MAPEO_TLOG_INVENTORY_TRANSFER.md, la condición de registro, la tabla de
// detalle y varios campos clave son [UNKNOWN]. Este generator retorna
// Empty=true hasta que LAGERBEW sea provisto y el mapeo sea completado.
//
// Ver MAPEO_TLOG_INVENTORY_TRANSFER.md para todos los [UNKNOWN] pendientes.
package transfer

import (
	"github.com/opessa/tlog-pipeline/internal/db"
	"github.com/opessa/tlog-pipeline/internal/naming"
	"github.com/opessa/tlog-pipeline/internal/tlog"
	"github.com/opessa/tlog-pipeline/internal/tlog/common"
)

// Generator implementa tlog.Generator para InventoryTransfer.
type Generator struct{}

func (Generator) Type() naming.TLOGType { return naming.TLOGTransfer }

// Generate retorna Empty=true porque la tabla LAGERBEW no está disponible.
// Cuando LAGERBEW sea incluida en el export, implementar el mapper aquí.
func (Generator) Generate(s *db.Store, h *common.HeaderCtx, kstID string) (*tlog.GenerateResult, error) {
	_ = s
	_ = h
	_ = kstID
	// LAGERBEW no está en los CSVs del proyecto. No se generan XMLs.
	return &tlog.GenerateResult{Empty: true}, nil
}
