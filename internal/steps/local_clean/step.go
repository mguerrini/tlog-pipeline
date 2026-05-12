// Package local_clean cierra el día borrando los artefactos intermedios
// locales. Si está habilitado se ejecuta siempre — no consulta el ftp_status.
// Conserva los XML generados y el propio ftp_status.json bajo
// target_root/AAAAMMDD, y antes de borrar copia los artefactos clave a
// finished_root/AAAAMMDD para trazabilidad.
package local_clean

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

func (Step) Name() string { return "local_clean" }

func (Step) Run(_ context.Context, d *pipeline.DayCtx) *pipeline.StepResult {
	b := pipeline.NewResult()

	if !d.Cfg.LocalClean.Enabled {
		return b.Skip("disabled in config")
	}

	dayStr := timeutil.FormatCompact(d.Day)

	// 1) Snapshot a finished_root/AAAAMMDD antes de limpiar (trazabilidad).
	copiedToFinished := 0
	finishedRoot := d.Cfg.LocalFolders.FinishedRoot
	if finishedRoot != "" {
		finishedDir := filepath.Join(finishedRoot, dayStr)
		if err := os.MkdirAll(finishedDir, 0o755); err != nil {
			return b.Fail(fmt.Errorf("crear finished_dir %s: %w", finishedDir, err))
		}
		statusFiles := []string{
			dayStr + "_day_status.json",
			dayStr + "_orphans.md",
			dayStr + "_pipeline.log",
			pipeline.FtpStatusFile,
		}
		for _, name := range statusFiles {
			src := filepath.Join(d.OutDir, name)
			if _, err := os.Stat(src); os.IsNotExist(err) {
				continue
			}
			if err := copyFile(src, filepath.Join(finishedDir, name)); err != nil {
				d.Log.Warn("no se pudo copiar archivo de status", "file", name, "err", err)
				continue
			}
			copiedToFinished++
		}
		csvCopied, err := copyDirContents(d.DayDir, finishedDir)
		if err != nil {
			d.Log.Warn("no se pudieron copiar inputs", "from", d.DayDir, "err", err)
		}
		copiedToFinished += csvCopied
	}

	// 2) Limpiar target_root/AAAAMMDD: borrar todo excepto .xml, ftp_status.json
	//    y — si delete_database=false — los .db. Si delete_database=true los
	//    archivos .db se borran junto con el resto de los intermedios.
	removed, err := cleanOutDir(d.OutDir, d.Cfg.LocalClean.DeleteDatabase)
	if err != nil {
		return b.Fail(fmt.Errorf("limpiar out_dir %s: %w", d.OutDir, err))
	}
	d.Log.Info("local_clean: out_dir limpiado", "dir", d.OutDir, "removed", removed)

	// 3) source_root/AAAAMMDD entero (input del día) — sólo si delete_source=true.
	if d.Cfg.LocalClean.DeleteSource {
		if err := os.RemoveAll(d.DayDir); err != nil {
			d.Log.Warn("no se pudo eliminar source dir", "dir", d.DayDir, "err", err)
		} else {
			d.Log.Info("local_clean: source dir eliminado", "dir", d.DayDir)
		}
	} else {
		d.Log.Info("local_clean: source dir conservado (delete_source=false)", "dir", d.DayDir)
	}

	b.SetMeta("copied_to_finished", copiedToFinished)
	b.SetMeta("removed_from_out", removed)
	return b.OK()
}

// cleanOutDir borra recursivamente los archivos bajo outDir excepto los .xml
// y el ftp_status.json. Los .db se borran sólo si deleteDB=true; sino se
// conservan junto con los .xml. Mantiene la estructura de directorios (no
// remueve subcarpetas, aunque queden vacías).
func cleanOutDir(outDir string, deleteDB bool) (int, error) {
	removed := 0
	err := filepath.Walk(outDir, func(p string, fi os.FileInfo, walkErr error) error {
		if walkErr != nil {
			if os.IsNotExist(walkErr) {
				return nil
			}
			return walkErr
		}
		if fi.IsDir() {
			return nil
		}
		name := filepath.Base(p)
		if name == pipeline.FtpStatusFile {
			return nil
		}
		ext := filepath.Ext(name)
		if strings.EqualFold(ext, ".xml") {
			return nil
		}
		if !deleteDB && strings.EqualFold(ext, ".db") {
			return nil
		}
		if err := os.Remove(p); err != nil {
			return fmt.Errorf("borrar %s: %w", p, err)
		}
		removed++
		return nil
	})
	return removed, err
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

// copyDirContents copia los archivos regulares (no recursivo) de srcDir a dstDir.
// Si srcDir no existe, retorna (0, nil).
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
