// Package split_by_date mueve los CSVs del día actual desde la carpeta única
// "all" hacia input/AAAAMMDD/ (la day_dir). El día se reconoce por los últimos
// 8 caracteres del basename del archivo.
//
// Se complementa con findDays, que también descubre días a partir de los
// nombres de archivo en "all/" — sino los días nuevos (sin carpeta input)
// nunca se procesarían.
package split_by_date

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/opessa/tlog-pipeline/internal/pipeline"
	"github.com/opessa/tlog-pipeline/internal/timeutil"
)

type Step struct{}

func (Step) Name() string { return "split_by_date" }

func (Step) Run(_ context.Context, d *pipeline.DayCtx) *pipeline.StepResult {
	b := pipeline.NewResult()
	if !d.Cfg.SplitByDate.Enabled {
		return b.Skip("disabled in config")
	}

	src := d.Cfg.LocalFolders.All
	if src == "" {
		return b.Skip("source vacío (local_folders.all)")
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		if os.IsNotExist(err) {
			return b.Skip("carpeta source no existe")
		}
		return b.Fail(fmt.Errorf("leer %s: %w", src, err))
	}

	if err := os.MkdirAll(d.DayDir, 0o755); err != nil {
		return b.Fail(fmt.Errorf("crear day_dir %s: %w", d.DayDir, err))
	}

	dayStr := timeutil.FormatCompact(d.Day)
	moved := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		ext := filepath.Ext(name)
		if !strings.EqualFold(ext, ".csv") {
			continue
		}
		base := strings.TrimSuffix(name, ext)
		if len(base) < 8 {
			continue
		}
		if base[len(base)-8:] != dayStr {
			continue
		}
		srcPath := filepath.Join(src, name)
		dstPath := filepath.Join(d.DayDir, name)
		if err := moveFile(srcPath, dstPath); err != nil {
			return b.Fail(fmt.Errorf("mover %s -> %s: %w", srcPath, dstPath, err))
		}
		moved++
	}

	b.SetMeta("moved", moved)
	b.SetMeta("src", src)
	d.Log.Info("split_by_date ok", "moved", moved, "src", src, "day_dir", d.DayDir)
	return b.OK()
}

// moveFile intenta rename (atómico, mismo volumen); si falla — típicamente
// cross-device — cae a copy + delete.
func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		return err
	}
	return os.Remove(src)
}
