// Package create_xml_sql implementa el step de generación de XMLs para el
// flujo SQL: lee la DB SQLite producida por create_sql_db, hidrata un db.Store
// equivalente y reusa los generators existentes.
// Solo se ejecuta si create_db.sql = true.
package create_xml_sql

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/opessa/tlog-pipeline/internal/naming"
	"github.com/opessa/tlog-pipeline/internal/pipeline"
	"github.com/opessa/tlog-pipeline/internal/sqldb"
	"github.com/opessa/tlog-pipeline/internal/timeutil"
	"github.com/opessa/tlog-pipeline/internal/tlog/common"
	"github.com/opessa/tlog-pipeline/internal/tlog/factory"
)

// Step implementa el step "create_xml_sql".
type Step struct{}

func (Step) Name() string { return "create_xml_sql" }

func (Step) Run(ctx context.Context, d *pipeline.DayCtx) *pipeline.StepResult {
	b := pipeline.NewResult()

	if !d.Cfg.CreateDB.SQL {
		return b.Skip("create_db.sql = false (flujo de archivo)")
	}
	if !d.Cfg.CreateXML.Enabled {
		return b.Skip("disabled in config")
	}

	if err := os.MkdirAll(d.OutDir, 0o755); err != nil {
		return b.Fail(fmt.Errorf("crear out_dir: %w", err))
	}

	dayStr := timeutil.FormatCompact(d.Day)
	dbPath := filepath.Join(d.OutDir, dayStr+"_pipeline.db")
	if _, err := os.Stat(dbPath); err != nil {
		return b.Fail(fmt.Errorf("DB SQLite no encontrada en %s (¿create_sql_db corrió?): %w", dbPath, err))
	}

	d.Log.Info("create_xml_sql: hidratando store desde DB SQLite", "db", dbPath)
	store, err := sqldb.LoadStore(dbPath)
	if err != nil {
		return b.Fail(fmt.Errorf("hidratar store desde sqlite: %w", err))
	}
	d.Store = store

	beginDT, err := timeutil.ApplyOffset(d.Day, d.Cfg.Process.BeginDateOffset)
	if err != nil {
		return b.Fail(fmt.Errorf("begin_date_offset inválido: %w", err))
	}
	endDT, err := timeutil.ApplyOffset(d.Day, d.Cfg.Process.EndDateOffset)
	if err != nil {
		return b.Fail(fmt.Errorf("end_date_offset inválido: %w", err))
	}

	kostst := store.Tables["KOSTST"]
	if kostst == nil {
		return b.Fail(fmt.Errorf("tabla KOSTST no encontrada en DB SQLite"))
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
			result, err := gen.Generate(store, h, kstID)
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
	b.SetMeta("source_db", dbPath)
	return b.OK()
}