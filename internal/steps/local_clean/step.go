package local_clean

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/opessa/tlog-pipeline/internal/pipeline"
	"github.com/opessa/tlog-pipeline/internal/timeutil"
)

// Step implementa el step "local_clean".
type Step struct{}

func (Step) Name() string { return "local_clean" }

func (Step) Run(ctx context.Context, d *pipeline.DayCtx) *pipeline.StepResult {
	b := pipeline.NewResult()
	if !d.Cfg.LocalClean.Enabled {
		return b.Skip("disabled in config")
	}

	finishedRoot := d.Cfg.LocalFolders.FinishedRoot
	if finishedRoot == "" {
		return b.Skip("finished_root no configurado")
	}

	dayStr := timeutil.FormatCompact(d.Day)
	finishedDir := filepath.Join(finishedRoot, dayStr)
	if err := os.MkdirAll(finishedDir, 0o755); err != nil {
		return b.Fail(fmt.Errorf("crear finished_dir %s: %w", finishedDir, err))
	}

	// Copiar archivos de status y orphans al finished
	statusFiles := []string{
		dayStr + "_day_status.json",
		dayStr + "_orphans.md",
		dayStr + "_pipeline.log",
	}
	moved := 0
	for _, name := range statusFiles {
		src := filepath.Join(d.OutDir, name)
		dst := filepath.Join(finishedDir, name)
		if _, err := os.Stat(src); os.IsNotExist(err) {
			continue
		}
		if err := copyFile(src, dst); err != nil {
			d.Log.Warn("no se pudo copiar status file", "file", name, "err", err)
		} else {
			moved++
		}
	}

	// Si delete_source, eliminar la carpeta source del día
	if d.Cfg.LocalClean.DeleteSource {
		if err := os.RemoveAll(d.DayDir); err != nil {
			d.Log.Warn("no se pudo eliminar source dir", "dir", d.DayDir, "err", err)
		} else {
			d.Log.Info("source dir eliminado", "dir", d.DayDir)
		}
	}

	b.SetMeta("files_moved_to_finished", moved)
	b.SetMeta("finished_dir", finishedDir)
	return b.OK()
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}
