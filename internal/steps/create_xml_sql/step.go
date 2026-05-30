// Package create_xml_sql implementa el step de generación de XMLs:
// abre la DB SQLite producida por create_sql_db y delega en los
// generators SQL de internal/tlogsql, que ejecutan queries directas.
package create_xml_sql

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"github.com/opessa/tlog-pipeline/internal/naming"
	"github.com/opessa/tlog-pipeline/internal/pipeline"
	"github.com/opessa/tlog-pipeline/internal/sequence"
	"github.com/opessa/tlog-pipeline/internal/timeutil"
	"github.com/opessa/tlog-pipeline/internal/tlog"
	"github.com/opessa/tlog-pipeline/internal/tlog/common"
	"github.com/opessa/tlog-pipeline/internal/tlogsql"
)

// Step implementa el step "create_xml_sql".
type Step struct{}

func (Step) Name() string { return "create_xml_sql" }

func (Step) Run(ctx context.Context, d *pipeline.DayCtx) *pipeline.StepResult {
	b := pipeline.NewResult()

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

	d.Log.Info("create_xml_sql: abriendo DB SQLite", "db", dbPath)
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return b.Fail(fmt.Errorf("abrir sqlite: %w", err))
	}
	defer conn.Close()

	beginDT, err := timeutil.ApplyOffset(d.Day, d.Cfg.Process.BeginDateOffset)
	if err != nil {
		return b.Fail(fmt.Errorf("begin_date_offset inválido: %w", err))
	}
	beginDT = beginDT.AddDate(0, 0, -1)
	endDT, err := timeutil.ApplyOffset(d.Day, d.Cfg.Process.EndDateOffset)
	if err != nil {
		return b.Fail(fmt.Errorf("end_date_offset inválido: %w", err))
	}

	retails, err := tlogsql.AllRetails(ctx, conn)
	if err != nil {
		return b.Fail(fmt.Errorf("listar retails: %w", err))
	}
	if len(retails) == 0 {
		return b.Fail(fmt.Errorf("no hay filas en KOSTST"))
	}

	namer := naming.DefaultNamer{IncludeDocumentType: d.Cfg.Process.FileNameIncludeDocumentType}
	generators := tlogsql.AllGenerators()
	totalXMLs := 0
	totalEmpty := 0
	// counters[TLOGType] guarda el próximo CONTADOR del SEQUENCENUMBER para
	// cada tipo de doc, compartido entre todos los KST_IDs del día.
	counters := make(map[naming.TLOGType]int)

	for _, retail := range retails {
		if retail.KstID == "" {
			continue
		}
		retailCode := common.FormatRetailStoreID(retail.KstCode)

		h := &common.HeaderCtx{
			BusinessDay:   d.Day,
			BeginDateTime: beginDT,
			EndDateTime:   endDT,
			OperatorID:    d.Cfg.Process.OperatorID,
			RetailStoreID: retailCode,
			WorkstationID: "0",
			Period:        "0",
			Subperiod:     "0",
			IsProduction:  d.Cfg.Process.IsProduction,
		}

		// Fase 1: pre-asignar SequenceNumbers para todos los docs del KST
		// (excepto Cierre, cuyo ListCandidateIDs devuelve nil).
		// El contador global se avanza aquí por tipo, antes de escribir
		// cualquier archivo, de modo que todos los docIDs del KST conocen
		// su seqNum antes de que comience la generación de XMLs.
		kstSeqMaps := make(map[naming.TLOGType]tlog.DocSeqMap)
		for _, gen := range generators {
			if !d.Cfg.Output.Enabled(gen.Type()) {
				continue
			}
			ids, err := gen.ListCandidateIDs(ctx, conn, retail.KstID)
			if err != nil {
				return b.Fail(fmt.Errorf("listar candidatos %s KST=%s: %w", gen.Type(), retail.KstID, err))
			}
			if len(ids) == 0 {
				continue
			}
			docNum := tlogDocNumber(gen.Type())
			sm := make(tlog.DocSeqMap, len(ids))
			for i, id := range ids {
				seqNum, err := sequence.Build(d.Day, docNum, counters[gen.Type()]+i)
				if err != nil {
					return b.Fail(fmt.Errorf("pre-asignar sequence %s KST=%s id=%s: %w", gen.Type(), retail.KstID, id, err))
				}
				sm[id] = seqNum
			}
			kstSeqMaps[gen.Type()] = sm
			counters[gen.Type()] += len(ids)
		}

		// Fase 2: generar XMLs usando los seqNums pre-asignados.
		// Cierre recibe seqMap=nil y usa startCounter directamente.
		for _, gen := range generators {
			if !d.Cfg.Output.Enabled(gen.Type()) {
				continue
			}
			seqMap := kstSeqMaps[gen.Type()] // nil para Cierre y Transfer
			var crossSeqMap tlog.DocSeqMap
			switch gen.Type() {
			case naming.TLOGReception:
				crossSeqMap = kstSeqMaps[naming.TLOGFiscalDocFC]
			case naming.TLOGFiscalDocFC:
				crossSeqMap = kstSeqMaps[naming.TLOGReception]
			case naming.TLOGReturn:
				crossSeqMap = kstSeqMaps[naming.TLOGFiscalDocNC]
			case naming.TLOGFiscalDocNC:
				crossSeqMap = kstSeqMaps[naming.TLOGReturn]
			}
			result, err := gen.Generate(ctx, conn, h, retail.KstID, seqMap, crossSeqMap, counters[gen.Type()])
			if err != nil {
				d.Log.Error("error generando TLOG SQL",
					"type", gen.Type(), "kst_id", retail.KstID, "err", err)
				continue
			}
			if result == nil || result.Empty || len(result.Files) == 0 {
				totalEmpty++
				d.Log.Info("tlog vacío", "type", gen.Type(), "kst_id", retail.KstID)
				continue
			}

			for _, f := range result.Files {
				filename := namer.XMLFile(gen.Type(), retailCode, f.SeqNum)
				outPath := filepath.Join(d.OutDir, filename)
				if err := os.WriteFile(outPath, []byte(f.XMLContent), 0o644); err != nil {
					return b.Fail(fmt.Errorf("escribir %s: %w", filename, err))
				}
				totalXMLs++
				d.Log.Info("xml generado",
					"file", filename, "lines", f.NumLines)
			}
			// Solo Cierre (seqMap==nil) avanza el contador en Fase 2;
			// los demás ya lo hicieron en Fase 1.
			if seqMap == nil {
				counters[gen.Type()] += result.NumDocs
			}
		}
	}

	b.SetMeta("xmls_generated", totalXMLs)
	b.SetMeta("tlogs_empty", totalEmpty)
	b.SetMeta("source_db", dbPath)
	return b.OK()
}

// tlogDocNumber mapea un TLOGType al DocumentNumber del package sequence.
func tlogDocNumber(t naming.TLOGType) sequence.DocumentNumber {
	switch t {
	case naming.TLOGReception:
		return sequence.DocReception
	case naming.TLOGReturn:
		return sequence.DocReturn
	case naming.TLOGTransfer:
		return sequence.DocTransfer
	case naming.TLOGAdjustment:
		return sequence.DocAdjustment
	case naming.TLOGCount:
		return sequence.DocCount
	case naming.TLOGFiscalDocFC:
		return sequence.DocFiscalDocFC
	case naming.TLOGFiscalDocNC:
		return sequence.DocFiscalDocNC
	default:
		return sequence.DocCierre
	}
}
