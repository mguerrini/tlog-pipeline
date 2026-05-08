package create_db

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/opessa/tlog-pipeline/internal/csvio"
	"github.com/opessa/tlog-pipeline/internal/db"
	"github.com/opessa/tlog-pipeline/internal/pipeline"
	"github.com/opessa/tlog-pipeline/internal/timeutil"
)

// Step implementa el step "create_db": carga CSVs en la store + chequeo de huérfanos.
type Step struct{}

func (Step) Name() string { return "create_db" }

func (Step) Run(ctx context.Context, d *pipeline.DayCtx) *pipeline.StepResult {
	b := pipeline.NewResult()
	if !d.Cfg.CreateDB.Enabled {
		return b.Skip("disabled in config")
	}
	if d.Cfg.CreateDB.SQL {
		return b.Skip("create_db.sql = true (flujo SQL)")
	}

	sep := d.Cfg.CreateDB.Separator
	if sep == "" {
		sep = ","
	}

	// Encontrar archivos CSV del día
	found, err := csvio.FindFiles(d.DayDir)
	if err != nil {
		return b.Fail(err)
	}

	store := db.NewStore()
	totalRows := 0

	// Cargar en orden de dependencias FK
	for _, tableName := range csvio.LoadOrder {
		path, ok := found[tableName]
		if !ok {
			d.Log.Warn("tabla no encontrada en CSVs", "table", tableName)
			continue
		}
		header, rows, err := csvio.Read(path, sep)
		if err != nil {
			return b.Fail(fmt.Errorf("leer %s: %w", path, err))
		}
		dbRows := make([]db.Row, len(rows))
		for i, m := range rows {
			dbRows[i] = db.Row(m)
		}
		store.AddTable(tableName, header, dbRows)
		totalRows += len(dbRows)
		d.Log.Info("tabla cargada", "table", tableName, "rows", len(dbRows))
	}

	store.BuildIndexes()
	d.Store = store

	b.SetMeta("rows_loaded", totalRows)
	b.SetMeta("tables_loaded", len(store.Tables))

	// Guardar snapshot opcional
	if d.Cfg.Process.KeepDBAfterRun {
		snapshotPath := filepath.Join(d.OutDir,
			fmt.Sprintf("%s_pipeline.db.json", timeutil.FormatCompact(d.Day)))
		if err := store.SaveSnapshot(snapshotPath); err != nil {
			d.Log.Warn("no se pudo guardar snapshot de DB", "err", err)
		}
	}

	// ── Chequeo de huérfanos ──────────────────────────────────────────────
	if err := os.MkdirAll(d.OutDir, 0o755); err != nil {
		d.Log.Warn("no se pudo crear out_dir para orphan report", "err", err)
	}
	orphanReport := db.RunOrphanCheck(store, d.Day, nil)
	orphanFile := filepath.Join(d.OutDir,
		fmt.Sprintf("%s_orphans.md", timeutil.FormatCompact(d.Day)))
	if err := db.WriteOrphanReportMD(orphanReport, orphanFile); err != nil {
		d.Log.Warn("no se pudo escribir orphan report", "err", err)
	}

	// Log por relación
	for _, res := range orphanReport.Results {
		rel := fmt.Sprintf("%s.%s->%s.%s",
			res.Relation.ChildTable, res.Relation.ChildCol,
			res.Relation.ParentTable, res.Relation.ParentCol)
		if res.CheckError != nil {
			d.Log.Error("orphan_check error", "relation", rel, "err", res.CheckError)
			continue
		}
		if res.RowsOrphan > 0 {
			d.Log.Warn("orphan_check",
				slog.String("relation", rel),
				slog.Int("rows", res.RowsTotal),
				slog.Int("orphans", res.RowsOrphan),
				slog.Int("distinct_orphans", res.DistinctOrphan))
		} else {
			d.Log.Info("orphan_check", "relation", rel, "rows", res.RowsTotal, "orphans", 0)
		}
	}
	d.Log.Info("orphan_summary",
		"status", orphanReport.OverallStatus,
		"relations_with_issues", orphanReport.CountOrphanRelations(),
		"file", filepath.Base(orphanFile))

	b.SetMeta("orphans_status", orphanReport.OverallStatus)
	b.SetMeta("orphans_relations_with_issues", orphanReport.CountOrphanRelations())
	b.SetMeta("orphans_total_rows", orphanReport.TotalOrphanRows())
	b.SetMeta("orphans_report_file", filepath.Base(orphanFile))

	return b.OK()
}
