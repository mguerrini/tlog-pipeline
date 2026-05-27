package read_files

import (
	"context"
	"fmt"
	"os"
	"strings"

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

	sep := d.Cfg.CreateDB.Separator
	if sep == "" {
		sep = "|"
	}
	for table, cols := range d.Cfg.ReadFiles.ClearCols {
		filePath, ok := found[table]
		if !ok {
			continue
		}
		if err := fixMultilineFile(filePath, sep, cols); err != nil {
			return b.Fail(fmt.Errorf("fix_multiline %s: %w", filePath, err))
		}
		d.Log.Info("read_files: columnas vaciadas", "table", table, "cols", cols)
	}

	b.SetMeta("files_found", len(found))
	b.SetMeta("day_dir", d.DayDir)
	d.Log.Info("read_files ok", "files_found", len(found), "day", d.Day.Format("20060102"))
	return b.OK()
}

// fixMultilineFile reconstruye un CSV cuyos campos contienen newlines sin comillas
// y vacía las columnas indicadas. Sobreescribe el archivo.
func fixMultilineFile(path, sep string, clearCols []string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		return nil
	}

	headerLine := strings.TrimRight(lines[0], "\r")
	targetSeps := strings.Count(headerLine, sep)
	if targetSeps == 0 {
		return fmt.Errorf("separador %q no encontrado en header de %s", sep, path)
	}

	headerCols := strings.Split(headerLine, sep)
	clearIdx := map[int]bool{}
	for _, col := range clearCols {
		for i, h := range headerCols {
			if strings.TrimSpace(h) == col {
				clearIdx[i] = true
			}
		}
	}

	var records []string
	accumulated := ""
	for _, line := range lines[1:] {
		line = strings.TrimRight(line, "\r")
		if accumulated == "" {
			accumulated = line
		} else {
			accumulated += "\n" + line
		}
		if strings.Count(accumulated, sep) == targetSeps {
			fields := strings.SplitN(accumulated, sep, targetSeps+1)
			for idx := range clearIdx {
				if idx < len(fields) {
					fields[idx] = ""
				}
			}
			records = append(records, strings.Join(fields, sep))
			accumulated = ""
		}
	}
	if accumulated != "" {
		return fmt.Errorf("registro incompleto al final de %s", path)
	}

	out := headerLine + "\n" + strings.Join(records, "\n") + "\n"
	return os.WriteFile(path, []byte(out), 0o644)
}
