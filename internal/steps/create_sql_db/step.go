// Package create_sql_db implementa el step de generación de DB SQLite tipada.
// Solo se ejecuta si create_db.sql = true en config.json (flujo SQL).
// Su salida (.db) la consume create_xml_sql para generar los XMLs.
package create_sql_db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/opessa/tlog-pipeline/internal/pipeline"
	"github.com/opessa/tlog-pipeline/internal/sqldb"
	"github.com/opessa/tlog-pipeline/internal/timeutil"
)

// Step implementa el step "create_sql_db".
type Step struct{}

func (Step) Name() string { return "create_sql_db" }

func (Step) Run(ctx context.Context, d *pipeline.DayCtx) *pipeline.StepResult {
	b := pipeline.NewResult()

	// Saltar si create_db.sql = false
	if !d.Cfg.CreateDB.SQL {
		return b.Skip("create_db.sql = false")
	}

	if err := os.MkdirAll(d.OutDir, 0o755); err != nil {
		return b.Fail(fmt.Errorf("crear out_dir: %w", err))
	}

	dayStr := timeutil.FormatCompact(d.Day)
	dbPath := filepath.Join(d.OutDir, dayStr+"_pipeline.db")
	reportPath := filepath.Join(d.OutDir, dayStr+"_sqldb_load.md")

	sep := d.Cfg.CreateDB.Separator
	if sep == "" {
		sep = ","
	}

	d.Log.Info("create_sql_db: generando DB SQLite tipada", "db", dbPath, "sep", sep)

	result, err := sqldb.Load(d.DayDir, dbPath, sep)
	if err != nil {
		return b.Fail(fmt.Errorf("sqldb.Load: %w", err))
	}

	// Escribir reporte Markdown (si logs.sql_db_load = true)
	if d.Cfg.Logs.SQLDBLoad {
		mdContent := sqldb.WriteReportMD(result, d.Day)
		if err := os.WriteFile(reportPath, []byte(mdContent), 0o644); err != nil {
			d.Log.Warn("no se pudo escribir reporte sqldb", "err", err)
		} else {
			d.Log.Info("reporte sqldb generado", "path", reportPath)
		}
	}

	// Loguear stats
	for _, s := range result.Stats {
		if s.Err != nil {
			d.Log.Error("error cargando tabla en sqldb",
				"table", s.Table, "err", s.Err)
		} else {
			d.Log.Info("tabla cargada en sqldb",
				"table", s.Table, "rows", s.Inserted, "dur", s.Duration)
		}
	}
	for _, o := range result.Orphans {
		if o.OrphanRows > 0 && o.ExpectedZero {
			d.Log.Error("FK huérfanas inesperadas", "relation", o.Label, "rows", o.OrphanRows)
		} else if o.OrphanRows > 0 {
			d.Log.Warn("FK huérfanas esperadas", "relation", o.Label, "rows", o.OrphanRows)
		}
	}

	b.SetMeta("db_path", dbPath)
	if d.Cfg.Logs.SQLDBLoad {
		b.SetMeta("report_path", reportPath)
	}
	b.SetMeta("overall_ok", result.OverallOK)

	// Contar totales para el meta
	totalInserted := 0
	for _, s := range result.Stats {
		totalInserted += s.Inserted
	}
	b.SetMeta("total_rows", totalInserted)

	if !result.OverallOK {
		d.Log.Warn("create_sql_db completado con advertencias — ver reporte")
	} else {
		d.Log.Info("create_sql_db completado OK", "rows", totalInserted, "db", dbPath)
	}

	return b.OK()
}
