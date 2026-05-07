package read_files

import (
	"context"
	"fmt"
	"os"

	"github.com/opessa/tlog-pipeline/internal/csvio"
	"github.com/opessa/tlog-pipeline/internal/pipeline"
)

// Step implementa el step "read_files".
type Step struct{}

func (Step) Name() string { return "read_files" }

func (Step) Run(ctx context.Context, d *pipeline.DayCtx) *pipeline.StepResult {
	b := pipeline.NewResult()

	if !d.Cfg.ReadFiles.Enabled {
		return b.Skip("disabled in config")
	}

	if err := os.MkdirAll(d.OutDir, 0o755); err != nil {
		return b.Fail(fmt.Errorf("crear out_dir %s: %w", d.OutDir, err))
	}

	found, err := csvio.FindFiles(d.DayDir)
	if err != nil {
		return b.Fail(err)
	}

	missing := []string{}
	for _, pattern := range d.Cfg.ReadFiles.ExpectedFiles {
		table := csvio.PatternToTable(pattern)
		if _, ok := found[table]; !ok {
			missing = append(missing, pattern)
		}
	}
	if len(missing) > 0 {
		return b.Fail(fmt.Errorf("archivos esperados no encontrados: %v", missing))
	}

	b.SetMeta("files_found", len(found))
	b.SetMeta("day_dir", d.DayDir)
	d.Log.Info("read_files ok", "files_found", len(found), "day", d.Day.Format("20060102"))
	return b.OK()
}
