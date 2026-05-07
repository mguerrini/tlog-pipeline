package tlog

import (
	"github.com/opessa/tlog-pipeline/internal/db"
	"github.com/opessa/tlog-pipeline/internal/naming"
	"github.com/opessa/tlog-pipeline/internal/tlog/common"
)

// GenerateResult devuelve metadata sobre la generación.
type GenerateResult struct {
	XMLContent string
	Empty      bool
	NumDocs    int
	NumLines   int
}

// Generator es la interfaz que implementan los 8 mappers.
type Generator interface {
	Type() naming.TLOGType
	Generate(s *db.Store, h *common.HeaderCtx, kstID string) (*GenerateResult, error)
}
