package tlog

import (
	"github.com/opessa/tlog-pipeline/internal/db"
	"github.com/opessa/tlog-pipeline/internal/naming"
	"github.com/opessa/tlog-pipeline/internal/tlog/common"
)

// GeneratedFile representa un archivo XML producido por un generador.
// Cada archivo contiene exactamente una <Transaction> identificada por SeqNum.
type GeneratedFile struct {
	SeqNum     string
	XMLContent string
	NumLines   int
}

// GenerateResult devuelve los archivos producidos y metadata agregada.
type GenerateResult struct {
	Files    []GeneratedFile
	Empty    bool
	NumDocs  int
	NumLines int
}

// Generator es la interfaz que implementan los 8 mappers.
//
// startCounter es el valor inicial del CONTADOR del SEQUENCENUMBER para esta
// llamada. El step llama a Generate una vez por KST_ID y va incrementando el
// contador con result.NumDocs, de modo que el SEQUENCENUMBER sea único por
// (día × tipo) a lo largo de todos los KST_IDs.
type Generator interface {
	Type() naming.TLOGType
	Generate(s *db.Store, h *common.HeaderCtx, kstID string, startCounter int) (*GenerateResult, error)
}
