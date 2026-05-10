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

	// Copiar archivos de entrada (CSVs del día) al finished
	sourceCopied, err := copyDirContents(d.DayDir, finishedDir)
	if err != nil {
		d.Log.Warn("no se pudieron copiar archivos de entrada", "from", d.DayDir, "err", err)
	}
	moved += sourceCopied

	// Si delete_source, eliminar la carpeta source del día
	if d.Cfg.LocalClean.DeleteSource {
		if err := os.RemoveAll(d.DayDir); err != nil {
			d.Log.Warn("no se pudo eliminar source dir", "dir", d.DayDir, "err", err)
		} else {
			d.Log.Info("source dir eliminado", "dir", d.DayDir)
		}
	}

	// Si delete_database, eliminar la BD SQLite generada en el output
	if d.Cfg.LocalClean.DeleteDatabase {
		dbPath := filepath.Join(d.OutDir, dayStr+"_pipeline.db")
		if err := os.Remove(dbPath); err != nil && !os.IsNotExist(err) {
			d.Log.Warn("no se pudo eliminar BD", "file", dbPath, "err", err)
		} else if err == nil {
			d.Log.Info("BD eliminada", "file", dbPath)
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

// copyDirContents copia los archivos regulares (no recursivo) de srcDir a dstDir.
// Devuelve la cantidad copiada. Si srcDir no existe, retorna (0, nil).
func copyDirContents(srcDir, dstDir string) (int, error) {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	copied := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if err := copyFile(filepath.Join(srcDir, e.Name()), filepath.Join(dstDir, e.Name())); err != nil {
			return copied, fmt.Errorf("copiar %s: %w", e.Name(), err)
		}
		copied++
	}
	return copied, nil
}
