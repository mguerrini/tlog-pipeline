package create_xml

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/opessa/tlog-pipeline/internal/naming"
	"github.com/opessa/tlog-pipeline/internal/pipeline"
	"github.com/opessa/tlog-pipeline/internal/tlog/common"
	"github.com/opessa/tlog-pipeline/internal/tlog/factory"
	"github.com/opessa/tlog-pipeline/internal/timeutil"
)

// Step implementa el step "create_xml".
type Step struct{}

func (Step) Name() string { return "create_xml" }

func (Step) Run(ctx context.Context, d *pipeline.DayCtx) *pipeline.StepResult {
	b := pipeline.NewResult()
	if !d.Cfg.CreateXML.Enabled {
		return b.Skip("disabled in config")
	}
	if d.Cfg.CreateDB.SQL {
		return b.Skip("create_db.sql = true (flujo SQL — usa create_xml_sql)")
	}
	if d.Store == nil {
		return b.Fail(fmt.Errorf("store no inicializada (create_db no corrió)"))
	}

	if err := os.MkdirAll(d.OutDir, 0o755); err != nil {
		return b.Fail(fmt.Errorf("crear out_dir: %w", err))
	}

	// Construir offsets del día
	beginDT, err := timeutil.ApplyOffset(d.Day, d.Cfg.Process.BeginDateOffset)
	if err != nil {
		return b.Fail(fmt.Errorf("begin_date_offset inválido: %w", err))
	}
	endDT, err := timeutil.ApplyOffset(d.Day, d.Cfg.Process.EndDateOffset)
	if err != nil {
		return b.Fail(fmt.Errorf("end_date_offset inválido: %w", err))
	}

	// Detectar retails activos en la store (todos los KST_ID presentes en KOSTST)
	kostst := d.Store.Tables["KOSTST"]
	if kostst == nil {
		return b.Fail(fmt.Errorf("tabla KOSTST no encontrada en store"))
	}

	namer := naming.DefaultNamer{}
	generators := factory.AllGenerators()
	totalXMLs := 0
	totalEmpty := 0

	for _, kstRow := range kostst.Rows {
		kstID := kstRow["KST_ID"]
		if kstID == "" {
			continue
		}
		retailCode := common.FormatRetailStoreID(kstRow["KST_CODE"])

		h := &common.HeaderCtx{
			BusinessDay:   d.Day,
			BeginDateTime: beginDT,
			EndDateTime:   endDT,
			OperatorID:    d.Cfg.Process.OperatorID,
			RetailStoreID: retailCode,
			WorkstationID: "0",
			Period:        "0",
			Subperiod:     "0",
		}

		for _, gen := range generators {
			if !d.Cfg.Output.Enabled(gen.Type()) {
				continue
			}
			result, err := gen.Generate(d.Store, h, kstID)
			if err != nil {
				d.Log.Error("error generando TLOG",
					"type", gen.Type(), "kst_id", kstID, "err", err)
				continue
			}
			if result == nil || result.Empty {
				totalEmpty++
				d.Log.Info("tlog vacío", "type", gen.Type(), "kst_id", kstID)
				continue
			}

			filename := namer.XMLFile(retailCode, d.Day, gen.Type())
			outPath := filepath.Join(d.OutDir, filename)
			if err := os.WriteFile(outPath, []byte(result.XMLContent), 0o644); err != nil {
				return b.Fail(fmt.Errorf("escribir %s: %w", filename, err))
			}
			totalXMLs++
			d.Log.Info("xml generado",
				"file", filename, "docs", result.NumDocs, "lines", result.NumLines)
		}
	}

	b.SetMeta("xmls_generated", totalXMLs)
	b.SetMeta("tlogs_empty", totalEmpty)
	return b.OK()
}


